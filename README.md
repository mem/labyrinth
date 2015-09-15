### The Go Challenge 6

#### Daedalus & Icarus

This program implements:

	* A maze generator
	* A maze solver
	* A client and server to exercise the generator and the solver

#### About this solution

This solution actually includes five maze generators:

	* A simple one that produces empty mazes
	* A simple one that produces topologially straight mazes
	* A ring maze generator
	* A binary tree generator
	* A general tree generator (from Mazes for programmers, by Jamis Buck)

The solver is a simple recursive solver, which works very well in Go.

Initially when I read the challenge's description I though of
implementing a concurrent maze solver, until I realized that the server
is not concurrent safe. I thought about changing the server to establish
a websocket connection for each solver and spawning multiple solvers on
the client side, but since the challenge's budget is not centered around
speed but around steps taken, I decided against that. That plus the fact
that I need to remain compatible with the standard server.

The solution has no tests whatsoever because I started exploring
solutions and implementations, and by the time I was happy with
something, I had dug myself into a test-unfriendly hole. I could have
written tests to verify that the mazes have solutions, but I know that
because the algorithms all generate perfect mazes.

Future directions? Implement more generators. Fix the server. Explore
different solvers. Figure out a better method to make the client's life
harder :-)
