Library for building chess variant engines

The top level function this library provides has this signature:

func Run(variant int, protocol int)

This call should run the engine in the specified variant ( like Racing Kings, Atomic etc. ) and using the specified protocol ( like uci, winboard etc. ).

All actual executables are created as separate packages under chessvariantengine which import this library and the only thing they do is that they call Run with the appropriate parameters.

For example the Racing Kings UCI engine is contained in the package github.com/chessvariantengine/racingkings/uci.