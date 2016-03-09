//////////////////////////////////////////////////////
// interface.go
// implements the engine interface
//////////////////////////////////////////////////////

package lib

// imports

import(
	"fmt"
	"os"
)

//////////////////////////////////////////////////////
// function Run is the entry point of the application
// it should be called with specifying the variant and the protocol
// all actual engines should be in a separate package
// the only thing they should do is to call this function
// this allows to have an standalone executable for every variant/protocol combination

func Run(variant int, protocol int) {
	Printu(fmt.Sprintf("variantengine %s %s by golang\n",VARIANT_TO_NAME[variant],PROTOCOL_TO_NAME[protocol]))
}

//////////////////////////////////////////////////////

// enumeration of variants

const(
	VARIANT_Standard          = iota             
	VARIANT_Racing_Kings
)

// enumeration of protocols

const(
	PROTOCOL_UCI              = iota
	PROTOCOL_XBOARD
)

// names of variants

var VARIANT_TO_NAME=[...]string{
	"Standard",
	"Racing Kings",
}

// names of protocols

var PROTOCOL_TO_NAME=[...]string{
	"UCI",
	"XBOARD",
}

///////////////////////////////////////////////
// Printu : unbuffered write to stdout
// -> str string : string to be written

func Printu(str string) {
	os.Stdout.Write([]byte(str))
}

///////////////////////////////////////////////