# Challenge #2: Unique ID Generation
https://fly.io/dist-sys/2/

## Single Node
A single node which generates UIDs is the simplest case. The node will keep
track of a counter in memory which it uses to assign a UID to each incoming
requests. The obvious issues with this approach will give us a concrete
place to begin:
1. It's not a distributed system (ha)
1. Node will redistribute IDs after it reboots, so the IDs aren't unique
1. Increased latency for concurrent requests due to contention for the counter
1. Impossible to scale horizontally because the current UID is managed by a node
1. The UID node is a single point of failure for anything dependent upon it
1. The number of available IDs is limited by the maximum value an integer type
can store in memory (implementation detail)

## Randomly Generated UIDs, Stored Centrally
Nodes will write generated UIDs to shared storage to coordinate which UIDs are
available. Nodes will regenerate UIDs if they generate an unavailable UID. 
Issues with this approach:
1. It's not a distributed system
1. Race condition when two nodes generate the same, available UID
1. The UID storage becomes a single point of failure
1. The number of available IDs is limited by the maximum value an integer type
can store in memory. A sufficiently large pool of UIDs is required to avoid
collisions.
Compariston to previous attempt:
1. Horizontally scalable, but may suffer from diminishing returns as competition
for central storage increases
1. The UID node is no longer a single point of failure

### Results
```
./maelstrom test -w unique-ids --bin /maelstrom/node --time-limit 5 --rate 10 --node-count 1 --availability total --nemesis partition
```
Ran a simple case to see if this is working as expected. No collisions were
detected, but that's expected with such a short test and a UID pool so large.
I expect this will work fine scaled up to three nodes. I'm a bit curious to see
what pool size will be sufficient to avoid failures. Of course, it all depends
on the test length. UID exhaustion with this strategy is an obvious flaw.

```
./maelstrom test -w unique-ids --bin /maelstrom/node --time-limit 30 --rate 1000 --node-count 3 --availability total --nemesis partition
```
Works decently well. Latency under 10ms for the most part. I didn't see any
collisions. I reduced the pool of available UIDs a few times to simulate the
gradual exhaustion of available IDs. This is a pretty big issue that makes this
solution untenable. I implemented some basic retry logic (yes, I know infinite
retries are dumb) and ran into the expected failures at the end of the test run.
The server became some busy with retries that new tests failed. I'll add a limit
to retries just to get the test to complete without errors and see how many UID
generations fail.