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
generations fail. Results are still awful

I didn't get around to implementing this idea, but my next thought was: why does
the generator need to store ALL the IDs indefinitely? this would solve the issue
of pool exhaustion. At the beginning of each day, we can clear the "used UID"
pool and forget about checking ALL the UIDs that have been generated since the
start of the system. The issue there is we need a way to distinguish groups of
IDs by day. To do that, a simple sequence number can be used that tracks what
day the UID was generated:

[ DAY : int ][ SEQUENCE NUMBER : int ]

This still relies on a central system to track the current pool of UIDs, which
isn't a distributed system. I knew time was an important aspect of the solution
to the UID problem because I read about UUIDs a few years ago (Twitter
snowflakes, specifically). But I couldn't remember the full solution.

Here's my favorite part:

I ended up posing this problem to my sister for fun. She came up with a
brilliant idea in just a few minutes. Each node can generate it's own set of
unique ideas using a sequence number and a node identifier. Her UIDs would
be of the form:

[ NODE ID : int ][ SEQUENCE NUMBER : int ]

My critique was that the management of the node IDs would be a chore; there
should never be a case where two nodes use the same ID. Managing that over
time with nodes that can come and go would be very annoying; one slip up and
you've got data corruption.

She was SO CLOSE to coming up with Twitter's snowflake ID and she has no
background in system design or software engineering. Truly amazing. We only
discussed the problem for maybe 15 minutes, tops.

I caved and looked up the ID format of Twitter's snowflakes the same night.

It became immediately apparent why their solution is so brilliant:
1. The milliseconds since epoch portion of their UID means that EVERY
MILLISECOND, there is a fresh pool of unique IDs, similiar to my day
refresh solution, but on a much shorter scale.
1. The sequence ID is similar to the basic, single node solution. The
sequence number is refreshed every millisecond.

At this point, we have an infinite number of UIDs to work with. The issue
remaining is: how do we do this in many different places without coordination?

The machine ID ties the two solutions together beautifully. Putting a machine
ID in the UID guarantees that each node will generate unique IDs as long as
their clock is locally consistent and no other node shares the same ID. This
does not mean however, that machine IDs must be universally unique. The
component of time means that machine IDs can be reused as long as two machines
with the same ID never exist at the same time. This eliminates the issue of
ID exhaustion based on the size of the machine ID portion of the UID.

The time portion of the UID ensures that there is always IDs available.
The machine ID guarantees that each machine will generate unique IDs.
And the sequence number defines the pool of available IDs each millisecond in
a stupidly simple way.

Absolutely brilliant. With a bit more refinement of our combined solutions, we
would have arrived at the solution. It's important to think about the attributes
of data that we can utilize and how they can form emergent properties when
combined. It's worthwhile to think through simple solutions because they still
have value, even when working on complex problems.

# Challenge #3: Broadcast
Broadcasts need to be more efficient. Initially, I would just fire and forget
a new broadcast to all neighbors of each node. Issues with this approach:
1. Loops will cause network flooding. Broadcasts will be propagated forever
1. Inefficient to send a message to all neighbors. Only neighbors who haven't
seen the message need to get it.

Point #2 is a bit interesting. With a fire and forget approach, it's not
possible to guarantee that a neighbor received a broadcast. However, each node
can track who sent it a message. The node can skip sending a message to a 
neighbor who already has it, because the neighbor was the one who sent it to
the node!

There is still a large issue, though. How does the system recover from a
partition? If a message is only broadcast once, while the network is
partitioned, how will the nodes in the partition without the sender get the
data? They can't request the data, because they don't know about it.

Is it possible for the nodes to detect a partition, then trigger recovery
procedures? Or perhaps a periodic check-in with other nodes is sufficient?

The former seems more resource efficient, but more complex. The latter is
simpler, but more resource inefficient.

## Challenge #3c: Fault Tolerant Broadcast
I did a little whiteboarding and realized I can use a handshake of sorts to
confirm that a message has been propagated to all neighbors! 

Here's the basics of the program
1. If the node hasn't seen the message before
    1. Initialize a map of message to neighbor on receipt of a new message
    1. Send all outstanding messages to truant neighbors
1. If the sender is not in the map for message
    1. Send the message back to the sender
1. Delete the sender from the map

<img src="./broadcast.svg">

The diagram shows the full exchange between two neighbors in the absence of a
partition. Now, consider partitions at each step where a message is in flight:
Step 2: The state of the graph will revert to Step 1. We can ignore this case
Step 4: n1 will send a message to n2 later. n2 will send the message back to n1
as confirmation. Eventually, the n1 will get the acknowledgement and stop
sending message A

The way I currently implemented this is very inefficient, especially with a
large amount of messages; outstanding messages will be sent out on every new
message. I need a mechanism which will send outstanding messages under specific
conditions that won't flood the network. In addition, the eventual consistency
guarantee only works if there is a constant flow of messages. Outstanding
messages will never make it if the flow of new messages stops!

## Challenge #4c: Efficient Broadcast: Part 1
Hmm. I think the maintainers of this website messed up. The next challange is
to make the broadcast system faster, but gives higher metrics for success...

I'll shuffle things around a bit to make more sense. In this part, I will
shoot for:

Messages-per-operation is below 30
Median latency is below 1 second
Maximum latency is below 2 seconds

---

Hoooooo boy. I need to get my messages per operation below 30... and they are currently at 7260... alrighty then.

I know FOR SURE that a big chunk of this is the silly logic I'm using to send
unacknowledged messages. I've gotta figure out a better strategy for that
before changing anything else.

Sending less than the entire set of outstanding messages is a good approach.
I'll use a queue of structs to send one outstanding message at a time, then
delete structs from the queue as acks come in.

---

The queue helped quite a bit. The implementation I used is crude and could
definitely be improved, but I'm going to ignore that for now. It may become
an issue when I get to the latency objectives. I reduced messages per 
operation down to 2800. Pretty substantial, but not nearly enough. Hmmm...

---

I think I've been trying to solve this problem with the false assumption that
I can only use the payloads which are described in the problem statement...

If I can construct my own payloads and even endpoints, this may become MUCH
easier.

Emergent properties (thanks Challenge #2!) may be important for efficient
propagation of messages. A node has some basic knowledge:
1. messages it has received
1. messages it has sent
1. which neighbor sent it a message
1. who its neighbors are

But there is also metadata available here:
1. when a message was received/sent
1. latency of an acknowledgement

With my own payloads/endpoints, I may be able to get even more valuable
metadata to make a better broadcast system!

Come to think of it, I had a great idea early on that I dropped because I
thought i was "against the rules": piggybacking broadcasts and acknowledgements!

---

Here's the plan: a node will send the entire list of outstanding messages for a
neighbor whenever a new message comes in (i.e. when we have to broadcast anyway)

Messages will be eventually acknowledged as long as there is a constant flow of
new messages.

If a receipient has already seen the message before and has already sent an
acknowledgement, it will queue up a response to the sender which will be sent
on the next broadcast.

So what we need is a payload like this:
``` go
{
    type: "broadcast",
    message: 0,
    acks: []int{1, 2, 3},
}
```

---

Internalizing the concept of "eventually consistent" made this problem much
easier to reason about, especially considering the # of message requirement.

In reality, a message only needs to be sent when there is a new message to
broadcast. Assuming there is a constant flow of new messages means that
acknowledgements and "inquiries" (when a node expected an acknowledgement,
but hasn't received it yet) can be delayed until a new message is
broadcast.

While nodes maintain their own internal state, but don't actually need to
communicate that to other nodes in the system. Specifically, each node will
track

1. the messages it has seen
1. the messages from each neighbor which needs to be acknowledged
1. the acknowledgements that it needs to send to each

neighbor. Only a list of messages is sent over the wire; it's up to the
receipient node to decide what to do with each message.

Each node can categorize messages into:

1. A message that is known and is expected
1. A message that is known and is unexpected
1. A message that is unknown

The second case is of particular interest; that is the case that determines
that the broadcast system is in a bad state. That bad state can be described
as, from the perspective of the receipient node "My neighbor is expecting
acknowledgement that I already sent." Lost acknowledgements put the system
into this state.

This diagram shows the state of neighboring nodes just after a message from
n1 to n0 was lost. The message contained the messages A, B, and C. As you can
see, node n1 is expecting acknowledgement of a message that n1 already
acknowledged; the n1 "ack" and n0 "expect" lists don't intersect.

<img src="./broadcast-bad-state.svg">

To recover, a node simply needs to add the unexpected message to the outgoing
acknowledgements and then wait for the next new message for the acknowledgement
to be sent! That's what n1 does when it sees A and B messages again.

Because the list of expected acknowledgments is only ever modified on receipt
of a message, dropped "inquiries" will never cause the system to enter into
a bad state. The dropped inquires will be resent next time a new message is
broadcast! This also includes every new message, new message are handled by this
as well.

Results:

```
:stable-latencies {
    0 0,
    0.5 449, <---- Missed the median by less than 50ms. Dang
    0.95 685,
    0.99 763,
    1 800 <--- Hmmm. 200ms off. Damn.
},
:net {
    :all {
        :send-count 60214,
        :recv-count 60214,
        :msg-count 60214,
        :msgs-per-op 29.676687
    },
    :clients {
        :send-count 4158,
        :recv-count 4158,
        :msg-count 4158
    },
    :servers {
        :send-count 56056,
        :recv-count 56056,
        :msg-count 56056,
        :msgs-per-op 27.627403 <--- WOW!!!!!!!!
    },
}

```

I missed the latency targets, but I then again I wasn't shooting to beat those
metrics on this pass. Pretty damn fine job if I do say so myself.

27 msgs-per-op down from the THOUSANDS before. Holy smokes!!!

Now I gotta figure out how to squeeze a bit more efficiency out of of my nodes.

As an adendum: I think I understand why this challenge has a lower threshold
than the next:

If we're aiming to decrease the messages per op by 1/3 in the next challenge,
latency will have to suffer because the latency is defined as:

"These latencies are measured from the time a broadcast request was acknowledged to when it was last missing from a read on any node."

Hmmm....