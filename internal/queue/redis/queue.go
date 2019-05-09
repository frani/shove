package redis

import (
	"context"
	"errors"
	"github.com/gomodule/redigo/redis"
	"gitlab.com/pennersr/shove/internal/queue"
	"gitlab.com/pennersr/shove/internal/types"
	"log"
	"time"
)

type redisQueueFactory struct {
	pool *redis.Pool
}

type redisQueue struct {
	id           string
	pool         *redis.Pool
	shuttingDown bool
}

func NewQueueFactory(url string) queue.QueueFactory {
	qf := &redisQueueFactory{
		pool: &redis.Pool{
			MaxIdle:     3,
			IdleTimeout: 240 * time.Second,
			Dial: func() (redis.Conn, error) {
				return redis.DialURL(url)
			},
		},
	}
	return qf
}

func (rq *redisQueue) Queue(msg types.PushMessage) (err error) {
	marshalled, err := msg.Marshal()
	if err != nil {
		return
	}
	conn := rq.pool.Get()
	defer conn.Close()
	l, _ := rq.listNames()
	_, err = conn.Do("RPUSH", l, marshalled)
	return nil
}

func (rq *redisQueue) listNames() (l, pl string) {
	l = "shove:" + rq.id
	pl = l + ":pending"
	return
}

func (rq *redisQueue) Shutdown() (err error) {
	rq.shuttingDown = true
	err = rq.pool.Close()
	return
}

func (rq *redisQueue) Remove(qm queue.QueuedMessage) (err error) {
	rqm := qm.(*redisQueuedMessage)
	conn := rq.pool.Get()
	defer conn.Close()
	_, pl := rq.listNames()
	n, err := redis.Int(conn.Do("LREM", pl, 1, rqm.raw))
	if err != nil {
		return
	}
	if n == 0 {
		log.Println("Push message already gone from pending list", pl)
	}
	return nil
}

func (rq *redisQueue) Requeue(qm queue.QueuedMessage) (err error) {
	rqm := qm.(*redisQueuedMessage)
	conn := rq.pool.Get()
	defer conn.Close()
	l, pl := rq.listNames()

	if err = conn.Send("MULTI"); err != nil {
		return
	}
	if err = conn.Send("LREM", pl, 1, rqm.raw); err != nil {
		return
	}
	if err = conn.Send("RPUSH", l, rqm.raw); err != nil {
		return
	}
	_, err = conn.Do("EXEC")
	return
}

func (rq *redisQueue) Get(ctx context.Context) (qm queue.QueuedMessage, err error) {
	conn := rq.pool.Get()
	defer conn.Close()
	l, pl := rq.listNames()

	var raw []byte
	for ctx.Err() == nil {
		raw, err = redis.Bytes(conn.Do("BRPOPLPUSH", l, pl, 2))
		if err == redis.ErrNil {
			err = nil
			continue
		}
		if err != nil {
			return
		}
		rqm := &redisQueuedMessage{raw: raw}
		rqm.msg, err = types.UnmarshalPushMessage(raw)
		if err == nil {
			qm = rqm
		}
		return
	}
	err = errors.New("queue shutting down")
	return
}

func (rq *redisQueue) recover() (err error) {
	conn := rq.pool.Get()
	defer conn.Close()
	l, pl := rq.listNames()
	for {
		_, err := redis.Bytes(conn.Do("RPOPLPUSH", pl, l))
		if err == redis.ErrNil {
			log.Println("No more", rq.id, "push notifications to recover")
			break
		}
		if err != nil {
			return err
		}
		log.Println("recovered pending", rq.id, "push notification")
	}
	return
}

func (rqf *redisQueueFactory) NewQueue(id string) (q queue.Queue, err error) {
	rq := &redisQueue{
		id:   id,
		pool: rqf.pool,
	}
	err = rq.recover()
	if err != nil {
		return
	}
	q = rq
	return
}