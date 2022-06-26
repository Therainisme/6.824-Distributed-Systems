# 6.824 分布式系统课程

[6.824 Schedule: Spring 2022](https://pdos.csail.mit.edu/6.824/schedule.html)

- [x] MapReduce: [Action](https://github.com/Therainisme/6.824-Spring-2022/runs/6915081108)
- [ ] Raft

## MapReduce

> Todo

## Raft 2A

论文中 Figure 2 的图已经描述得非常明确了。这里有实现的一些细节。

这里必须要实现一个状态机，**当切换状态后，之前的所有执行的函数必须退出。**，而且锁的颗粒度应该尽可能的大，但不能锁上 RPC。

实验中的 RPC Call，如果锁上 Call 了，若模拟的网络延时非常大，调用方将会一直阻塞在 Call 调用处。什么时候返回 true 和 false，在 RequestVote 的注释中写得很清楚。

> A false return can be caused by a dead server, a live server that can't be reached, a lost request, or a lost reply.

```go
ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
```

最最最重要的一个大坑就是：

> any goroutine with a long-running loop should call killed() to check whether it should stop.

> 任何一个长期运行循环的 goroutine 必须调用 `killed()` 检查该 peer 是否被 kill 掉了，然后选择是否应该停止。

如果我没有记错的话，

```go
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}
```

### 状态机？

![image](https://user-images.githubusercontent.com/41776735/175816741-60f7b806-50ad-4585-9a27-7e17cf15a7c3.png)

大致写一下详细的细节吧！

#### Follower

Follower 默认拥有一个定时器 Timer

1. 收到 RequestVote，重置 Timer
2. 收到 AppendEntries，重置 Timer
3. 如果 Timer 到期，进入 Candidate

#### Candidate

Candidate 默认拥有一个定时器 Timer（与 Follower 相同）

1. 收到超过半数 Granted，直接进入 Leader 状态
2. 收到 Leader 的 AppendEntries，直接进入 Follower 状态
3. 收到 Candidate 的 RequestVote，拒绝投票，保持 Candidate
4. 如果 Timer 到期，未进行状态变更，则进入下一阶段的选举，保持 Candidate

#### Leader

Leader 默认拥有一个定时器 Timer，但是设定的时间比 Follower 和 Candidate 还要小。

1. 如果 Timer 到期，向所有 Server 发送 AppendEntries

#### 所有 Server 必须进行状态降级的条件

如果 RPC Request 或 Response 中的 Term，比自己的 currentTerm 大，则设置自己的 currentTerm 为请求中的 Term，并装换状态为 Follower。

#### 如何在状态切换的时候停止执行上一个状态的函数？

> Todo