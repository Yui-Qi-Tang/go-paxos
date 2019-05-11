# go-paxos

I had previously studied the Paxos algorithm, but had not implement it, which is why this project exists.

我曾經研究過Paxos演算法，但我不曾實做過它，這就是這個專案為什麽會存在的原因。

Thanks to the open course from MIT http://nil.csail.mit.edu/6.824/2015/labs/lab-3.html I learned a lot of techiques from the lab-3 and enabled my implementation of the Paxos algorithm.

很感謝MIT的公開課程，我在lab-3學到很多技巧並讓我有能力去實做它。

Thanks!

感謝**Keviv**大大幫忙修正我的英文文法XDDD

Go version: 1.12.5

*There are so many go rpc complains during testing, just ignore!

測試時可能會有很多來自go rpc的抱怨，目前先忽略吧！

HINT:

    RPC node unix FD is created in paxos-nodes/

Run test:

    run all tests: go test -v