# 自己的总结

摘要：Raft 将一致性的特征分解：选举、日志复制和安全性等，还提供一种更改集群成员的新机制。它比 Paxos 更容易理解，更简单。

## 5. The Raft consensus algorithm

Raft 通过选举一个高效的 Leader 实现一致性。Client 只能与 Leader 进行通信。Leader 与 Follower 的数据流向也是单向的，只允许 Leader 向 Follower 进行 AppendEntries。Leader 接收 Client 的日志条目，然后复制给其他的服务器。当这些日志条目可以安全的写入的时，Leader 会告诉这些服务器进行写入。如果 Leader 断连，则进行选举。

### 5.1 Raft basics

Raft 能容忍不超过一半的 Server 故障。例如有 5 个服务器，Raft 最多能容忍 2 个服务器故障。每一个时刻 Server 只能有三种状态，Leader、Candidate 和 Follower。Follower 是消极的，它不会发出任何的请求，只会简单地相应 Leader 或 Candidate 的 RPC。

Raft 将时间划分成不同长度的 Term。一个 Term 开始于一个 Candidate 的选举。相当于皇朝的建立。皇朝不能同时有两个君主。同理，每个 Term 只能有一个 Leader。

但每个 Server 可能在同一时间看到的是不同的 Term。每一个 Server 中会储存 Term，只增不减。Server 之间的 RPC 请求和响应都会带上自己的 Term，如果对方的 Term 比自己的大，则更新自己的 Term。（如果是 Leader 或 Candidate 还需要转变为 Follower）如果对方的 Term 比自己的小，则拒绝该 RPC 请求。

Raft 实现一致性需要两个最基础的 RPC 方法，RequestVote 和 AppendEntries。前者用于 Candidate 进行选举时的投票，后者用于 Leader 向其他 Follower 发送日志复制请求和心跳包的发送。在第七章快照的实现中，还会提供第三个 RPC 方法。

### 5.2 Leader election

当一个 Server 启动是，默认是 Follower。如果他不断地收到 Candidate 的 RequestVote 或 Leader 的 AppendEntries，则会一直保持 Follower 的状态。如果一段时间没有收到上述两个 RPC，就认为现在没有一个正常工作的 Leader，则会进行选举。

Leader 为了防止其他的 Follower 变成 Candidate，会定期的发送心跳包。Candidate 为了防止其他的 Follower 变成 Candidate，会发送 RequestVote RPC 请求。

当一个 Follower 进行选举，它将自己的 Term + 1，转变成 Candidate 状态。并行的进行 RequestVote 请求。当发生这三件事情时，Candidate 状态会结束：

1. 它自己赢得了选举
2. 另一个 Server 告诉它：“我是 Leader”（当然这个 Leader 的 Term ≥ Candidate 的 Term。因为 RequestVote 只能拒绝，所以 Leader 想告诉 Candidate “我是 Leader” 时，只能发送 AppendEntries 心跳包）
3. 投票结束了，没有选出 Leader

当 Candidate 收到了大多数的投票后（count > len(servers)）则迅速进入 Leader 状态，开始发送 AppendEntries。这个选举测量保证一次选举中只能选出一个 Leader。

Raft 选择一个 150~300ms 的超时定时器进入选举状态，以防止选举失败的情况出现（范围足够大的话，同一时间进入选举状态的只有一个 Server）。