package browser

import (
	"context"
	"crypto/rand"
	"log/slog"
	"os"
	"regexp"
	"sync"
	"time"
)

// 任务状态。
const (
	jobQueued  = "queued"
	jobRunning = "running"
	jobDone    = "done"
	jobFailed  = "failed"
)

const (
	// maxPendingJobs 是排队上限，超过才回「忙」；正常使用（一次存一整页几十张）远到不了。
	maxPendingJobs = 500
	// keptJobs 是保留查询结果的已完成任务数，再旧的扔掉。
	keptJobs = 1000
)

type job struct {
	id      string
	request Request
	ctx     context.Context
	state   string
	err     error
	done    chan struct{}
}

// jobQueue 让扩展的导入一个接一个做：先返回任务号，Momento 自己排队，不再回「忙」。
type jobQueue struct {
	run      ImportFunc
	mu       sync.Mutex
	jobs     map[string]*job
	finished []string
	pending  []*job
	working  bool
}

func newJobQueue(run ImportFunc) *jobQueue {
	return &jobQueue{run: run, jobs: map[string]*job{}}
}

// submit 排进队列。ctx 是服务这次运行的上下文：关掉扩展服务时，没做完的任务一起取消。
func (q *jobQueue) submit(ctx context.Context, request Request) (*job, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.pending) >= maxPendingJobs {
		return nil, false
	}
	j := &job{id: rand.Text(), request: request, ctx: ctx, state: jobQueued, done: make(chan struct{})}
	q.jobs[j.id] = j
	q.pending = append(q.pending, j)
	if !q.working {
		q.working = true
		go q.loop()
	}
	return j, true
}

func (q *jobQueue) loop() {
	for {
		q.mu.Lock()
		if len(q.pending) == 0 {
			q.working = false
			q.mu.Unlock()
			return
		}
		j := q.pending[0]
		q.pending = q.pending[1:]
		j.state = jobRunning
		q.mu.Unlock()

		started := time.Now()
		err := j.ctx.Err()
		if err == nil {
			err = q.run(j.ctx, j.request)
		}
		if j.request.File != "" {
			// 上传的临时文件导入时会被挪走；没挪走（失败、取消）就在这里删掉。
			if removeErr := os.Remove(j.request.File); removeErr != nil && !os.IsNotExist(removeErr) {
				slog.Warn("删除上传的临时文件失败", "error", removeErr)
			}
		}
		code, _ := classify(err)
		q.mu.Lock()
		j.err = err
		j.state = jobDone
		if err != nil {
			j.state = jobFailed
		}
		q.finished = append(q.finished, j.id)
		for len(q.finished) > keptJobs {
			delete(q.jobs, q.finished[0])
			q.finished = q.finished[1:]
		}
		q.mu.Unlock()
		if err != nil {
			slog.Warn("浏览器扩展导入任务失败", "job", j.id, "code", code, "duration", time.Since(started), "error", redactURLs(err.Error()))
		} else {
			slog.Info("浏览器扩展导入任务完成", "job", j.id, "duration", time.Since(started))
		}
		close(j.done)
	}
}

// view 是任务当前的状态，给查询接口用。
func (q *jobQueue) view(id string) (jobView, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	j, ok := q.jobs[id]
	if !ok {
		return jobView{}, false
	}
	v := jobView{ID: j.id, State: j.state}
	if j.err != nil {
		v.Code, _ = classify(j.err)
		v.Error = j.err.Error()
	}
	return v, true
}

// 下载报错里常带完整网址（含签名参数），写日志前换掉。
var urlPattern = regexp.MustCompile(`(?i)\b[a-z][a-z0-9+.-]*://[^\s"']+`)

func redactURLs(text string) string {
	return urlPattern.ReplaceAllString(text, "[url]")
}
