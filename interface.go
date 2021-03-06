//////////////////////////////////////////////////////
// interface.go
// implements the engine interface
// zurichess sources: main.go, uci.go
//////////////////////////////////////////////////////

///////////////////////////////////////////////
//  : 
// ->  : 
// <-  : 

///////////////////////////////////////////////

package lib

// imports

import(
	"fmt"
	"os"
	"bufio"
	"log"
	"strings"
	"time"
	"bytes"
	"regexp"
	"strconv"
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"sort"
)

//////////////////////////////////////////////////////
// function Run is the entry point of the application
// it should be called with specifying the variant and the protocol
// all actual engines should be in a separate package
// the only thing they should do is to call this function
// this allows to have an standalone executable for every variant/protocol combination

var uci *UCI

func Run(variant int, protocol int, bookblob *[]byte) {
	// set current variant
	Variant = variant
	// set current protocol
	Protocol = protocol
	// set book json blob
	BookJsonBlob = bookblob

	ClearLog()

	ClearBook()

	LoadSimpleBook(bookblob)

	/*if protocol == PROTOCOL_XBOARD {
		UseBook = true
		LoadBook()
	}*/

	// create uci
	uci = NewUCI()

	// initialize uci to current variant
	uci.SetVariant(VARIANT_CURRENT)

	// print introduction
	if Protocol == PROTOCOL_UCI {
		Printu(Intro())
	}

	// set up logging
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Lshortfile)

	// command interpreter main loop
	scan := bufio.NewScanner(os.Stdin)

	for scan.Scan() {
		scannedline := scan.Text()
		err := ExecuteLine(scannedline)
		if err == errQuit {
			break
		}
	}

	if scan.Err() != nil {
		log.Println(scan.Err())
	}
}

//////////////////////////////////////////////////////

///////////////////////////////////////////////
// definitions

// log commands in log.txt
var DO_LOG                    = true

// enumeration of XBOARD states
const(
	XBOARD_Initial_State      = iota
	XBOARD_Observing
	XBOARD_Analyzing
	XBOARD_Analysis_Complete
	XBOARD_Waiting
	XBOARD_Thinking
	XBOARD_Pondering
	XBOARD_Ponder_Complete
)

var XBOARD_State_Names = [...]string{
	"Initial State",
	"Observing",
	"Analyzing",
	"Analysis_Complete",
	"Waiting",
	"Thinking",
	"Pondering",
	"Ponder_Complete",
}

// XBOARD state
// https://chessprogramming.wikispaces.com/Chess+Engine+Communication+Protocol
var XBOARD_State              = XBOARD_Initial_State

// XBOARD side which the engine has to play
var XBOARD_Engine_Side        = Black

// XBOARD post mode
var XBOARD_Post               = true

// XBOARD level number of moves per block
var XBOARD_level_moves        = 40

// XBOARD level time [millisecond]
var XBOARD_level_time         = 300000

// XBOARD level increment [millisecond]
var XBOARD_level_increment    = 0

// XBOARD time [millisecond]
var XBOARD_time               = 300000

// XBOARD otim [millisecond]
var XBOARD_otim               = 300000

// XBOARD do hint
var XBOARD_do_hint            = false

// enumeration of variants
const(
	VARIANT_Standard          = iota             
	VARIANT_Racing_Kings
	VARIANT_Atomic
	VARIANT_Horde
)

// variant flags
var(
	IS_Standard bool          = false
	IS_Racing_Kings bool      = false
	IS_Atomic bool            = false
	IS_Horde bool             = false
)

// side having only pawns in horde
var HORDE_Pawns_Side          = White

// side having normal pieces in horde
var HORDE_Pieces_Side         = HORDE_Pawns_Side.Opposite()

// starting positions for variants
var START_FENS = [...]string{
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		"8/8/8/8/8/8/krbnNBRK/qrbnNBRQ w - - 0 1",
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		"rnbqkbnr/pppppppp/8/1PP2PP1/PPPPPPPP/PPPPPPPP/PPPPPPPP/PPPPPPPP w kq - 0 1",
	}

// current variant
const VARIANT_CURRENT        = -1

// line read from stdin for execution
var line string

// line args
var args []string

// number of args
var numargs int

// argument pointer
var argptr int

// line command
var command string

// Author
var Author = "Alexandru Mosoi"

// current variant
var Variant int

// current protocol
var Protocol int

// test mode
var TEST bool = true

// use unicode symbols in test print of board
var USE_UNICODE_SYMBOLS = false

// enumeration of protocols
const(
	PROTOCOL_UCI              = iota
	PROTOCOL_XBOARD
)

// names of variants
var VARIANT_TO_NAME=[...]string{
	"Standard",
	"Racing Kings",
	"Atomic",
	"Horde",
}

// names of protocols
var PROTOCOL_TO_NAME=[...]string{
	"UCI",
	"XBOARD",
}

// variant name to variant
var VARIANT_NAME_TO_VARIANT=map[string]int{
	"Standard": VARIANT_Standard,
	"Racing Kings": VARIANT_Racing_Kings,
	"Atomic": VARIANT_Atomic,
	"Horde": VARIANT_Horde,
}

var VARIANT_SHORTHAND_NAME_TO_VARIANT=map[string]int{
	"s": VARIANT_Standard,
	"rk": VARIANT_Racing_Kings,
	"a": VARIANT_Atomic,
	"h": VARIANT_Horde,
}

// variant and protocol to engine name
type EngineNameIndex struct{
	variant int
	protocol int
}

var VARIANT_AND_PROTOCOL_TO_ENGINE_NAME=map[EngineNameIndex]string{
	EngineNameIndex{ variant: VARIANT_Standard, protocol: PROTOCOL_UCI }:"zurichess",
	EngineNameIndex{ variant: VARIANT_Racing_Kings, protocol: PROTOCOL_UCI }:"verkuci",
	EngineNameIndex{ variant: VARIANT_Racing_Kings, protocol: PROTOCOL_XBOARD }:"verkxboard",
	EngineNameIndex{ variant: VARIANT_Atomic, protocol: PROTOCOL_UCI }:"venatuci",
	EngineNameIndex{ variant: VARIANT_Atomic, protocol: PROTOCOL_XBOARD }:"venatxboard",
	EngineNameIndex{ variant: VARIANT_Horde, protocol: PROTOCOL_UCI }:"vehoruci",
	EngineNameIndex{ variant: VARIANT_Horde, protocol: PROTOCOL_XBOARD }:"vehorxboard",
}

// quit application 'error'
var (
	errQuit = fmt.Errorf("quit")
)

// test command ok 'error'
var (
	errTestOk = fmt.Errorf("testok")
)

// uciLogger outputs search in uci format.
type uciLogger struct {
	start time.Time
	buf   *bytes.Buffer
}

// UCI implements uci protocol
type UCI struct {
	Engine      *Engine
	timeControl *TimeControl

	// buffer of 1, if empty then the engine is available
	ready chan struct{}
	// buffer of 1, if filled then the engine is pondering
	ponder chan struct{}
	// predicted position hash after 2 moves
	predicted uint64
}

var MakeAnalyzedMove bool = false

// book definitions

// store scores option
var StoreScores bool = true

// minimal depth required for storing a score
var StoreMinDepth = 11

// book version : should be stored together with engine score in the book
// if the engine is modified a higher book version can signal
// that scores with lower version number should be overwritten
var BookVersion int = 2

// book move entry holds the evaluation of single move
type BookMoveEntry struct {
	// move in algebraic notation
	Algeb string
	// score returned by search
	Score int
	// depth of search that returned Score
	Depth int
	// book version
	BookVersion int
	// HasEval tells whether move has a minimaxed eval
	HasEval bool
	// eval determined by minimaxing
	Eval int
	// nodes used to calculate this eval ( this is book nodes )
	Nodes int
	// reserved for future use
	Int1 int // annotation
	Int2 int
	Int3 int
	Str1 string
	Str2 string
	Str3 string
}

// moventries type
type BookMoveEntries map[string]BookMoveEntry

// book entry holds all evaluated moves for a given position
type BookPositionEntry struct {
	// fen
	Fen string
	// move entries
	// key for move entries is move in algebraic notation
	MoveEntries BookMoveEntries
}

// book main entry holds the entire book
type BookMainEntry struct {
	// position entries
	// key for position entries is the Zobrist key converted to string
	PositionEntries map[string]BookPositionEntry
}

// book
var Book BookMainEntry

// simple book
var SimpleBook map[string]string

// ignore moves
var IgnoreMoves = []Move{}

// random number generator
var Rand=rand.New(rand.NewSource(time.Now().UnixNano()))

// dont print pv
var DontPrintPV = false

// max book depth
const MAX_BOOK_DEPTH = 48

// use book instead of search where available
var UseBook = false

// flag indicating that a book has been loaded
var BookLoaded = false

// count miminaxing
var MinimaxCnt = 0

// save book after certain number of minimaxes
var SaveBookAfterMinimaxCnt = 5

// book building under way
var BookBuildingUnderWay = false

// multipv mode
var MultiPV = 1

// current multipv index
var MultiPVIndex = 1

// multipv item holds search information on a single pv line
type MultiPVItem struct {
	Stats Stats
	Score int32
	Line []Move
	AlgebLine []string
	InfoString string
}

// multipv list holds all the pv items of the search
type MultiPVItemList []MultiPVItem

// list of all multipv items
var MultiPVList = MultiPVItemList{}

// current search depth
var CurrentSearchDepth int32

// latest available score
var LastScore int32

// cutoff limit for book building in centipawns
var BookCutOff int32 = 3000

// probability limits for selecting nodes in book building in the function of depth
var SelectLimits = [MAX_BOOK_DEPTH]int{
	90, // 1
	60, // 2
	30, // 3
	10, // 4
	10, // 5
	10, // 6
	10, // 7
	10, // 8
	10, // 9
	10, // 10
	10, // 11
	10, // 12
	10, // 13
	10, // 14
	10, // 15
	10, // 16
	10, // 17
	10, // 18
	10, // 19
	10, // 20
	10, // 21
	10, // 22
	10, // 23
	10, // 24
	10, // 25
	10, // 26
	10, // 27
	10, // 28
	10, // 29
	10, // 30
	10, // 31
	10, // 32
	10, // 33
	10, // 34
	10, // 35
	10, // 36
	10, // 37
	10, // 38
	10, // 39
	10, // 40
	10, // 41
	10, // 42
	10, // 43
	10, // 44
	10, // 45
	10, // 46
	10, // 47
	10, // 48
}

// add move chan
var AddMoveChan chan int

// minimax max depth
var MinimaxMaxDepth = 0

// book json blob
var BookJsonBlob *[]byte = nil

// minimum nodes required for move in simple book
var MinSimpleBookNodes = 3

// max number of legal moves per position
var MAX_LEGAL_MOVES = 500

// end definitions
///////////////////////////////////////////////

///////////////////////////////////////////////
// HasScore : multipv item list has at least one item that can provide a score
// -> mpvl *MultiPVItemList : multipv item list
// <- book : true if has score

func (mpvl *MultiPVItemList) HasScore() bool {
	if len(*mpvl) <= 0 {
		return false
	}
	return true
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// GetScore : get the score from multipv item list
// -> mpvl *MultiPVItemList : multipv item list
// <- int32 : best score in the list

func (mpvl *MultiPVItemList) GetScore() int32 {
	if !mpvl.HasScore() {
		// we should not get here
		return 0
	}
	mpvl.Sort()
	return (*mpvl)[0].Score
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// MultiPVItemList sort interface

func (ml *MultiPVItemList) Sort() {
	sort.Sort(ml)
}

func (ml *MultiPVItemList) Len() int {
	return len(*ml)
}

func (ml *MultiPVItemList) Less(i,j int) bool {
	return (*ml)[i].Score > (*ml)[j].Score
}

func (ml *MultiPVItemList) Swap(i,j int) {
	(*ml)[i] , (*ml)[j] = (*ml)[j] , (*ml)[i]
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// CreateMultiPVItem : creates a multipv item
// -> ul *uciLogger : uci logger
// -> stats Stats : engine stats
// -> line []Move : line
// <- MultiPVItem : created multipv item

func (ul *uciLogger) CreateMultiPVItem(stats Stats, score int32, line []Move) MultiPVItem {
	item := MultiPVItem{
		Stats : stats,
		Score : score,
		Line : line,
	}
	item.AlgebLine=[]string{}
	for _, m := range item.Line {
			item.AlgebLine=append(item.AlgebLine,m.UCI())
		}
	item.InfoString=ul.ReportInfoString(item)
	return item
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// ZobristStr : get the Zobrist key of the position as string
// -> pos *Position : position
// <- string : Zobrist key as string

func (pos *Position) ZobristStr() string {
	return fmt.Sprintf("%d",pos.Zobrist())
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// GetBookEntry : get the book entry for position
// -> pos *Position : position
// <- BookPositionEntry : book position entry
// <- bool : true if position is in the book

func (pos *Position) GetBookEntry() ( BookPositionEntry , bool ) {
	posentry , found := Book.PositionEntries[pos.ZobristStr()]
	return posentry , found
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// GetEval : get eval of move entry
// returns eval if it has minimaxed eval, otherwise return the engine search score
// -> mentry *BookMoveEntry : move entry
// <- int : eval

func (mentry *BookMoveEntry) GetEval() int {
	if mentry.HasEval {
		return mentry.Eval
	}
	return mentry.Score
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// GetSortedMoveEntryList : get sorted move entry list for position entry
// -> bentry *BookPositionEntry : position entry
// <- []BookMoveEntry : sorted move entry list

func (posentry *BookPositionEntry) GetSortedMoveEntryList() []BookMoveEntry {
	mentrylist := []BookMoveEntry{}
	for _ , mentry := range posentry.MoveEntries {
		// sort
		inserted := false
		for i := 0 ; i < len(mentrylist) ; i++ {
			greater := ( mentry.GetEval() > mentrylist[i].GetEval() )
			if ( mentry.Int1 != 0 ) && ( mentrylist[i].Int1 != 0 )	{
				// both moves annotated
				if mentry.Int1 != mentrylist[i].Int1 {
					// overrule only if annotations differ
					greater = mentry.Int1 > mentrylist[i].Int1
				}
			} else if ( mentry.Int1 != 0 ) {
				// first move is annotated
				greater = mentry.Int1 > 0
			} else if ( mentrylist[i].Int1 != 0 ) {
				// second move is annotated
				greater = mentrylist[i].Int1 < 0
			}
			if greater {
				sorted := []BookMoveEntry{}
				for j := 0 ; j < i ; j++ {
					sorted = append(sorted, mentrylist[j])
				}
				sorted = append(sorted, mentry)
				for j := i ; j < len(mentrylist) ; j++ {
					sorted = append(sorted, mentrylist[j])
				}
				mentrylist = sorted
				inserted = true
				break
			}
		}
		if !inserted {
			mentrylist = append(mentrylist, mentry)
		}
	}
	return mentrylist
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// GetSortedMoveEntryList : get the list of move entries in book entry for position
// the list is sorted in descending order of eval
// -> pos *Position : position
// <- []BookMoveEntry : move entry list

func (pos *Position) GetSortedMoveEntryList() []BookMoveEntry {
	posentry , found := pos.GetBookEntry()
	if !found {
		return []BookMoveEntry{}
	}
	mentrylist := posentry.GetSortedMoveEntryList()
	return mentrylist
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// GetBookMoveList : get the list of moves in book entry for position
// the list is sorted in descending order of eval
// -> pos *Position : position
// <- []Move : move list

func (pos *Position) GetBookMoveList() []Move {
	movelist := []Move{}
	mentrylist := pos.GetSortedMoveEntryList()
	for i := 0 ; i < len(mentrylist) ; i++ {
		m , err := pos.UCIToMove(mentrylist[i].Algeb)
		if err == nil {
			movelist = append(movelist, m)
		}
	}
	return movelist
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// SignedScore : signed version of score as string
// -> score int : score
// <- string : signed score

func SignedScore(score int) string {
	if score <= 0 {
		return fmt.Sprintf("%d",score)
	}
	return fmt.Sprintf("+%d",score)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// ToPrintable : move entry in printable form
// -> mentry *BookMoveEntry : move entry
// -> pos *Position : if position is not nil the LAN notation will be used
// <- string : move entry printable

func (mentry *BookMoveEntry) ToPrintable(pos *Position) string {
	evalstr := SignedScore(mentry.GetEval())
	mstr := mentry.Algeb
	if pos != nil {
		move , err := pos.UCIToMove(mstr)
		if err == nil {
			mstr = move.LAN()
		}
	}
	return fmt.Sprintf(" ( %3s , %5d ) %8s %5s", SignedScore(mentry.Int1), mentry.Nodes, mstr, evalstr)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// BookLineMoves : calculates book line moves for position
// -> pos *Position : position
// <- []Move : line

func (pos *Position) BookLineMoves() []Move {
	mentrylist := pos.GetSortedMoveEntryList()
	line := []Move{}
	cnt := 0
	for ( len(mentrylist) > 0 ) && ( cnt <= MAX_BOOK_DEPTH ) {
		algeb := mentrylist[0].Algeb
		move , err := pos.UCIToMove(algeb)
		if err == nil {
			cnt++
			line = append(line,move)
			pos.DoMove(move)
			mentrylist = pos.GetSortedMoveEntryList()
		} else {
			for i := 0 ; i < cnt ; i++ {
				pos.UndoMove()
			}
			return line
		}
	}
	for i := 0 ; i < cnt ; i++ {
		pos.UndoMove()
	}
	return line
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// CalcLine : calculate line from move list
// -> pos *Position : position
// -> moves []Move : move list
// <- string : line

func (pos *Position) CalcLine(moves []Move) string {
	if len(moves) <= 0 {
		return "*"
	}
	line := ""
	for i , move := range moves {
		algeb := move.UCI()
		pos.DoMove(move)
		if i==0 {
			line = algeb
		} else {
			line += " "+algeb
		}
	}
	for range moves {
		pos.UndoMove()
	}
	return line
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// BookMovesToPrintable : printable version of book moves for position
// -> pos *Position : position
// <- string : printable version of book moves

func (pos *Position) BookMovesToPrintable() string {
	pentry , found := pos.GetBookEntry()
	buff := fmt.Sprintf("book moves for position ( book size %d positions ) :", len(Book.PositionEntries))
	if !found {
		buff += " <none>\n"
		return buff
	}
	buff += "\n"
	cnt := 0
	for _ , mentry := range pentry.GetSortedMoveEntryList() {
		algeb := mentry.Algeb
		move , err := pos.UCIToMove(algeb)
		if err == nil {
			if cnt < 20 {
				mep := mentry.ToPrintable(pos)
				pos.DoMove(move)
				buff += fmt.Sprintf("%2d.%s %s\n", cnt+1, mep, pos.CalcLine(pos.BookLineMoves()))
				pos.UndoMove()
				cnt ++
			}
		}
	}
	return buff
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// GetMoveEntry : get a move entry from a position entry
// -> pentry *BookPositionEntry : position entry
// -> algeb string : move in algebraic notation
// <- BookMoveEntry : book move entry
// <- bool : true if move was found in the position entry

func (pentry *BookPositionEntry) GetMoveEntry(algeb string) ( BookMoveEntry , bool ) {
	mentry , found := pentry.MoveEntries[algeb]
	return mentry , found
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// GetMoveEntry : get the book move entry for move
// -> pos *Position : position
// -> algeb string : move in algebraic notation
// <- BookMoveEntry : book move entry
// <- bool : true if move is in the book

func (pos *Position) GetMoveEntry(algeb string) ( BookMoveEntry , bool ) {
	pentry , pfound := pos.GetBookEntry()
	if !pfound {
		return BookMoveEntry{} , false
	}
	return pentry.GetMoveEntry(algeb)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// DeletePositionEntryMoves : delete moves in a position entry
// -> pos *Position : position

func (pos *Position) DeletePositionEntryMoves() {
	Book.PositionEntries[pos.ZobristStr()] = BookPositionEntry{
		MoveEntries : make(BookMoveEntries),
	}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// StoreMoveEntry : store move entry for move
// -> pos *Position : position
// -> algeb string : move in algebraic notation
// -> mentry BookMoveEntry : move entry

func (pos *Position) StoreMoveEntry(algeb string, mentry BookMoveEntry) {
	pentry , pfound := pos.GetBookEntry()
	if !pfound {
		pentry = BookPositionEntry{
			MoveEntries : make(BookMoveEntries),
		}
	}
	pentry.MoveEntries[algeb] = mentry
	pentry.Fen = pos.String()
	Book.PositionEntries[pos.ZobristStr()] = pentry
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// ClearBook : creates Book as an empty book

func ClearBook() {
	Book.PositionEntries = make(map[string]BookPositionEntry)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// SaveBook : saves Book to disk

func SaveBook() {
	f,err:=os.Create("book.txt")
	if err!=nil {
		panic(err)
	} else {
		//b , err := json.MarshalIndent(Book, "", "    ")
		b , err := json.Marshal(Book)
		if err != nil {
			panic(err)
		}
		f.Write(b)
		f.Close()
	}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// SaveSimpleBook : saves simple book to disk

func SaveSimpleBook() {
	uci.SetVariant(VARIANT_CURRENT)
	PrintBookPage()
	MinimaxOutVerbose()
	f,err:=os.Create("simplebook.txt")
	if err!=nil {
		panic(err)
	} else {
		simplebook := make(map[string]string)
		poscnt := 0
		for zobriststr, posentry := range Book.PositionEntries {
			mentrylist := posentry.GetSortedMoveEntryList()
			if len(mentrylist) > 0 {
				bestentry := mentrylist[0]
				if bestentry.Nodes >= MinSimpleBookNodes {
					simplebook[zobriststr] = bestentry.Algeb
					poscnt++
				}
			}
		}
		fmt.Printf("number of positions %d\n", poscnt)
		b , err := json.Marshal(simplebook)
		if err != nil {
			panic(err)
		}
		f.Write(b)
		f.Close()
	}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// SaveBookVerbose : saves Book to disk and reports it

func SaveBookVerbose() {
	fmt.Printf("saving book ... ")
	SaveBook()
	fmt.Printf("done\n")
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// SaveBookAuto : saves Book to disk if book has been loaded from disk before

func SaveBookAuto() {
	if BookLoaded {
		SaveBook()
	}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// SaveBookAuto : saves Book to disk if book has been loaded from disk before and reports it

func SaveBookAutoVerbose() {
	if BookLoaded {
		fmt.Printf("auto ")
		SaveBookVerbose()
	}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// LoadBookVerbose : loads Book from disk and reports is

func LoadBookVerbose() {
	fmt.Printf("loading book ... ")
	LoadBook()
	fmt.Printf("done\n")
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// LoadBook : loads Book from disk

func LoadBook() {
	jsonBlob , err := ioutil.ReadFile("book.txt")
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(jsonBlob, &Book)
	if err != nil {
		panic(err)
	}
	BookLoaded = true
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// LoadSimpleBook : loads simple book

func LoadSimpleBook(bookblob *[]byte) {
	if bookblob == nil {
		SimpleBook = make(map[string]string)
		return
	}
	err := json.Unmarshal(*bookblob, &SimpleBook)
	if err != nil {
		panic(err)
	}
	//fmt.Printf("simple book size %d positions\n", len(SimpleBook))
}

///////////////////////////////////////////////


///////////////////////////////////////////////
// IsBookCutOff : check is score is a book cutoff
// -> score int32 : score
// <- bool : true if cutoff

func IsBookCutOff(score int32) bool {
	return ( ( score < -BookCutOff ) || ( score > BookCutOff ) )
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// AddNodeRecursive : add node recursive
// -> depth int : depth
// <- bool : true if node was added

func AddNodeRecursive(depth int, line string) bool {
	if depth >= MAX_BOOK_DEPTH {
		return false
	}
	pos := uci.Engine.Position
	mentrylist := pos.GetSortedMoveEntryList()
	if len(mentrylist) <= 0 {
		return AddMove(line)
	} else if IsBookCutOff(int32(mentrylist[0].Score)) {
		return false
	} else {
		for _ , mentry := range mentrylist {
			algeb := mentry.Algeb

			r := Rand.Intn(100)
			limit := SelectLimits[depth]

			numlegals := len(pos.GetLegalMoves(GET_ALL))

			if numlegals > 0 {

				fairshare := 100 / numlegals

				share := 100 - limit

				if share < fairshare {
					limit = 100 - fairshare
				}

			}

			randok := ( r > limit )

			version := mentry.BookVersion
			versionok := ( version <= BookVersion )

			score := int32(mentry.Score)
			scoreok := ( !IsBookCutOff(score) )

			selectok := ( randok && scoreok )

			if !versionok {
				pos.DeletePositionEntryMoves()
				break
			} else if selectok {
				move , err := uci.Engine.Position.UCIToMove(algeb)
				if err == nil {
					uci.Engine.DoMove(move)
					res := AddNodeRecursive(depth+1, line+" "+algeb)
					uci.Engine.UndoMove()
					return res
				}
			}
		}

		return AddMove(line)
	}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// MinimaxOut : minimax out book wrt current position recursively
// -> depth int : depth
// -> line []uint64 : line in Zobrist keys
// <- int : eval

var MinimaxNodes = 0

func MinimaxOutRecursive(depth int, line []uint64) int {
	MinimaxNodes++
	if depth > MinimaxMaxDepth {
		MinimaxMaxDepth = depth
	}
	alpha := int(-InfinityScore)
	if depth >= MAX_BOOK_DEPTH {
		return alpha
	}
	pos := uci.Engine.Position
	zobrist := pos.Zobrist()
	for _, z := range line {
		if zobrist == z {
			return 0
		}
	}
	pentry , found := pos.GetBookEntry()
	if found {
		for algeb , mentry := range pentry.MoveEntries {
			score := mentry.Score
			move , err := pos.UCIToMove(algeb)
			if err == nil {
				startnodes := MinimaxNodes
				uci.Engine.DoMove(move)
				eval := -MinimaxOutRecursive(depth+1, append(line, zobrist))
				if eval == int(InfinityScore) {
					eval = score
				}
				mentry.Eval = eval
				mentry.HasEval = true
				nodes := MinimaxNodes - startnodes
				mentry.Nodes = nodes
				if eval > alpha {
					alpha = eval
				}
				pentry.MoveEntries[algeb] = mentry
				uci.Engine.UndoMove()
			}
		}
		Book.PositionEntries[pos.ZobristStr()] = pentry
	}
	return alpha
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// MinimaxOut : minimax out book wrt current position
// <- int : eval

func MinimaxOut() int {
	MinimaxNodes = 0
	MinimaxMaxDepth = 0
	return MinimaxOutRecursive(0,[]uint64{})
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// MinimaxOutVerbose : minimax out book wrt current position and report it
// <- int : eval

func MinimaxOutVerbose() int {
	fmt.Printf("minimaxing out ( no %d ) ... ", MinimaxCnt)
	eval := MinimaxOut()
	fmt.Printf("done ( nodes %d maxdepth %d )\n", MinimaxNodes, MinimaxMaxDepth)
	return eval
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// PrintBookPage : print book page

func PrintBookPage() {
	fmt.Print(uci.Engine.Position.BookMovesToPrintable())
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// BuildBook : build book, should be run in go routine

var BuildBookStopped bool

var BuildBookReady chan int

func BuildBook() {
	DontPrintPV = true
	trycnt := 0
	okcnt := 0
	totalokcnt := 0
	for !BuildBookStopped {
		trycnt++
		if AddNodeRecursive(0,"*") {
			okcnt++
			totalokcnt++
			if okcnt >= 10 {
				MinimaxCnt++
				fmt.Println()
				MinimaxOutVerbose()
				PrintBookPage()
				if ( MinimaxCnt % SaveBookAfterMinimaxCnt ) == 0 {				
					SaveBookAutoVerbose()
				}
				okcnt = 0
			}
		}
		// if at least one node cannot be added per 1000 tries something is wrong, better to finish
		if ( trycnt >= 1000 ) && ( ( totalokcnt*1000 ) < trycnt ) {
			fmt.Printf("\n\nwarning: add move fails too much, book building auto stopped\n\n")
			break	
		}
	}
	BuildBookReady <- 0
	DontPrintPV = false
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// StartBuildBook : start building book

func StartBuildBook() {
	BuildBookReady = make(chan int)
	BuildBookStopped = false
	BookBuildingUnderWay = true
	go BuildBook()
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// StopBuildBook : stop building book

func StopBuildBook() {
	BuildBookStopped = true
	<- BuildBookReady
	MinimaxOutVerbose()
	PrintBookPage()
	SaveBookAutoVerbose()
	fmt.Printf("book building stopped\n")
	BookBuildingUnderWay = false
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// AddMove : add move to current position's book moves
// -> line string : line to which move is added
// <- bool : true if move was added

func AddMove(line string) bool {
	pos := uci.Engine.Position

	_ , final := uci.Engine.EndPosition()

	if final {
		return false
	}

	IgnoreMoves = pos.GetBookMoveList()

	LegalMoves := pos.GetLegalMoves(GET_ALL)

	if len(LegalMoves) <= 0 {
		return false
	}

	if len(IgnoreMoves) >= len(LegalMoves) {
		// if all moves were already searched, nothing to do
		return false
	}

	command := fmt.Sprintf("go depth %d", StoreMinDepth)

	GlobalHashTable.Clear()

	AddMoveChan = make(chan int)

	ExecuteLine(command)

	// wait for analysis to finish
	<- AddMoveChan

	if MultiPVList.HasScore() {
		score := MultiPVList.GetScore()
		if len(line) > 50 {
			line = line[0:49] + " ..."
		}
		fmt.Printf("\r   %-60s %s          \r", line + " " + MultiPVList[0].AlgebLine[0], SignedScore(int(score)))
		return true
	}

	return false
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// AddMoveUpTo : add book moves to position until input move is added
// -> san string : algeb
// <- bool : true if move was added

func AddMoveUpTo(san string) bool {
	pos := uci.Engine.Position
	move, err := pos.SANToMove(san)
	if err !=nil {
		return false
	}
	algeb := move.UCI()
	added := true
	DontPrintPV = true
	for added {
		posentry, found := pos.GetBookEntry()
		if found {
			_, mfound := posentry.MoveEntries[algeb]
			if mfound {
				DontPrintPV = false
				return true
			}
		}
		added = AddMove("+")
	}
	DontPrintPV = false
	return false
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// ClearLog : create an empty log.txt file

func ClearLog() {
	if !DO_LOG {
		return
	}
	// for debugging purposes
	f,err:=os.Create("log.txt")
		if err!=nil {
			panic(err)
		} else {
			f.Close()
		}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// Log : append string to log.txt
// -> what string : string to be appended

func Log(what string) {
	if !DO_LOG {
		return
	}
	// for debugging purposes
	f,err:=os.OpenFile("log.txt",os.O_CREATE|os.O_APPEND|os.O_WRONLY,0666)
	if err!=nil {
	    panic(err)
	}

	defer f.Close()

	if _,err=f.WriteString(what); err!=nil {
	    panic(err)
	}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// ExecuteLine : execute command line
// <- error : error

func ExecuteLine(setline string) error {
	line = strings.TrimSpace(setline)
	// print command line to log
	Log(fmt.Sprintf("-> %s\n",line))
	args = strings.Fields(line)
	var err error = nil
	if len(args)>0 {
		// only try to execute a line that has at least one token
		// first token is the command
		command = args[0]
		// rest are the arguments
		args = args[1:]	
		numargs = len(args)
		argptr = 0
		// first look at test commands
		if TEST {
			err = ExecuteTest()
		}
		// if test did not handle the command execute it by protocol
		if err == nil { switch Protocol {
			case PROTOCOL_UCI: err = ExecuteUci()
			case PROTOCOL_XBOARD : err = ExecuteXboard()
		}}
		if err != nil {
			if err != errQuit && err != errTestOk {
				if Protocol == PROTOCOL_UCI {
					log.Println(err)
				}
			}
		}
	}
	return err
}

///////////////////////////////////////////////
// ExecuteTest : execute TEST command
// <- error : error

func ExecuteTest() error {
	if line == "m" {
		MakeAnalyzedMove = true
		return ExecuteLine("stop")
	}
	switch command {
		case "x": return errQuit
		case "p":
			uci.PrintBoard()
			return errTestOk
		case "s":
			return ExecuteLine("stop")
		case "t":
			uci.SetVariant(VARIANT_Standard)
			uci.PrintBoard()
			return errTestOk
		case "r":
			uci.SetVariant(VARIANT_Racing_Kings)
			uci.PrintBoard()
			return errTestOk
		case "a":
			if numargs == 2 {
				if AnnotateMove() == nil {
					PrintBookPage()
				}
			} else if numargs == 1 {
				if AddMoveUpTo(args[0]) {
					fmt.Println()
					PrintBookPage()
				} else {
					fmt.Printf("adding move failed\n")
				}
			} else {
				uci.SetVariant(VARIANT_Atomic)
				uci.PrintBoard()
			}
			return errTestOk
		case "h":
			uci.SetVariant(VARIANT_Horde)
			uci.PrintBoard()
			return errTestOk
		case "intro":
			fmt.Print(Intro())
			return errTestOk
		case "uu":
			USE_UNICODE_SYMBOLS=true
			return errTestOk
		case "uc":
			USE_UNICODE_SYMBOLS=false
			return errTestOk
		case "m":
			uci.MakeSanMove(line)
			return errTestOk
		case "d":
			uci.UndoMove(line)
			return errTestOk
		case "l":
			uci.Engine.Position.PrintLegalMoves()
			return errTestOk
		case "pb":
			PrintBookPage()
			return errTestOk
		case "vs":
			PrintPieceValues()
			return errTestOk
		case "sb":
			SaveBook()
			return errTestOk
		case "ssb":
			SaveSimpleBook()
			return errTestOk
		case "lb":
			LoadBook()
			PrintBookPage()
			return errTestOk
		case "an":
			AddNodeRecursive(0,"*")
			return errTestOk
		case "bb":
			StartBuildBook()
			return errTestOk
		case "mo":
			MinimaxOutVerbose()
			return errTestOk
		case "q":
			if BookBuildingUnderWay {
				StopBuildBook()
			} else {
				LoadBookVerbose()
				PrintBookPage()
				StartBuildBook()
			}
			return errTestOk
		case "bs":
			StopBuildBook()
			return errTestOk
		case "sv":
			if numargs>0 {
				ok := false
				setvariant := GetRest()
				variant,found := VARIANT_NAME_TO_VARIANT[setvariant]
				if found {
					Variant = variant
					ok = true
				}
				variant,found = VARIANT_SHORTHAND_NAME_TO_VARIANT[setvariant]
				if found {
					Variant = variant
					ok = true
				}
				if ok {
					uci.SetVariant(VARIANT_CURRENT)
					fmt.Printf("variant set to %s\n",VARIANT_TO_NAME[Variant])
					return errTestOk
				} else {
					fmt.Printf("unknown variant %s\n",args[0])
					return errTestOk
				}
			} else {
				fmt.Printf("current variant %s\n",VARIANT_TO_NAME[Variant])
				return errTestOk
			}
	}
	return nil
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// ExecuteUci : execute UCI command
// <- error : error

func ExecuteUci() error {
	return uci.Execute(line)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// ExecuteUci : execute XBOARD command
// <- error : error

func ExecuteXboard() error {
	//Log(fmt.Sprintf("received command %s in state %s\n",line,XBOARD_State_Names[XBOARD_State]))
	// state independent commands
	if command != "undo" {
		XBOARD_undo_cnt = 0
	}
	switch command {
	case "quit":
		// quit applies to all XBOARD states
		return errQuit
	case "option":
		return uci.XBOARD_option()
	case "force":
			err := uci.XBOARD_force()
			if err != nil {
				return err
			}
			XBOARD_State = XBOARD_Observing
			return nil
	case "go":
		err := uci.XBOARD_go()
		if err != nil {
			return err
		}
		if XBOARD_State == XBOARD_Observing {
			return uci.XBOARD_Start_Thinking()
		}
		return nil
	case "hint":
		XBOARD_do_hint = true
		return uci.XBOARD_Start_Thinking()
	case "level":
		return uci.XBOARD_level()
	case "time":
		return uci.XBOARD_time()
	case "otim":
		return uci.XBOARD_otim()
	case "post":
		return uci.XBOARD_post()
	case "nopost":
		return uci.XBOARD_nopost()
	case "undo":
		err := uci.XBOARD_undo()
		if err != nil {
			return err
		}
		return nil
	case "setboard":
		return uci.XBOARD_setboard()
	case "analyze":
		err := uci.XBOARD_analyze()
		if err != nil {
			return err
		}
		XBOARD_State = XBOARD_Analyzing
		XBOARD_Check_Analyze()
		return nil
	}
	switch XBOARD_State {
	case XBOARD_Initial_State:
		switch command {
		case "xboard":
			Printu(fmt.Sprintf("feature myname=\"%s by Alexandru Mosoi\""+
			" analyze=1 variants=\"atomic\""+
			" option=\"UseBook -button\""+
			" setboard=1 usermove=1 playother=1 done=1\n",GetEngineName()))
			XBOARD_State = XBOARD_Observing
			return nil
		}
	case XBOARD_Observing:
		switch command {
		case "usermove":
			err := uci.XBOARD_usermove()
			if err != nil {
				return err
			}
			return nil
		case "new":
			err := uci.XBOARD_new()
			if err != nil {
				return err
			}
			XBOARD_State = XBOARD_Waiting
			return nil
		case "playother":
			err := uci.XBOARD_playother()
			if err != nil {
				return err
			}
			XBOARD_State = XBOARD_Pondering
			return nil
		}
	case XBOARD_Analyzing:
		switch command {
		case "usermove":
			err := uci.XBOARD_usermove()
			if err != nil {
				return err
			}
		case "exit":
			err := uci.XBOARD_exit()
			if err != nil {
				return err
			}
			XBOARD_State = XBOARD_Observing
			return nil
		}
	case XBOARD_Analysis_Complete:
		switch command {
		case "exit":
			err := uci.XBOARD_exit()
			if err != nil {
				return err
			}
			XBOARD_State = XBOARD_Observing
			return nil
		}
	case XBOARD_Waiting:
		switch command {
		case "usermove":
			err := uci.XBOARD_usermove()
			if err != nil {
				return err
			}
			return uci.XBOARD_Start_Thinking()
		}
	case XBOARD_Thinking:
		// if engine sends the 'move' command
		// state should change to XBOARD_Pondering
	case XBOARD_Pondering:
		switch command {
		case "usermove":
			err := uci.XBOARD_usermove()
			if err != nil {
				return err
			}
			return uci.XBOARD_Start_Thinking()
		}
	case XBOARD_Ponder_Complete:
		switch command {
		case "usermove":
			err := uci.XBOARD_usermove()
			if err != nil {
				return err
			}
			return uci.XBOARD_Start_Thinking()
		}
	}
	return nil
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// XBOARD commands

///////////////////////////////////////////////
// XBOARD_Error : format and prints an XBOARD error and returns it
// -> etype string : error type
// -> evalue string : error value

func XBOARD_Error(etype, evalue string) error {
	estr := fmt.Sprintf("Error (%s): %s", etype, evalue)
	fmt.Printf("%s\n", estr)
	return fmt.Errorf(estr)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// XBOARD_option : XBOARD option command
// -> uci *UCI : UCI
// <- error : error

func (uci *UCI) XBOARD_option() error {
	if numargs < 1 {
		return XBOARD_Error("wrong number of arguments for option",fmt.Sprintf("%d",numargs))
	}
	option := args[0]
	switch option {
	case "UseBook":
		UseBook = true
		//Log("use book accepted\n")
	}
	return nil
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// XBOARD_Check_Analyze : check if analysis should start upon changing the position

func XBOARD_Check_Analyze() {
	// start analyzing move
	if XBOARD_State == XBOARD_Analyzing {
		//Log("check analyze, stopping uci\n")

		uci.stop("")
		
		uci.timeControl = NewTimeControl(uci.Engine.Position, false)
		uci.timeControl.MovesToGo = 30 // in case there is not time refresh
		
		uci.timeControl.Start(false)
		uci.ready <- struct{}{}

		//Log("check analyze, starting analysis\n")

		IgnoreMoves = []Move{}

		go uci.play()
	}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// XBOARD_usermove : XBOARD usermove command
// -> uci *UCI : UCI
// <- error : error

func (uci *UCI) XBOARD_usermove() error {
	//Log(fmt.Sprintf("received usermove command with %d args\n",numargs))
	if numargs != 1 {
		return XBOARD_Error("wrong number of arguments for usermove",fmt.Sprintf("%d",numargs))
	}
	// stop any ongoing analysis
	//Log("stopping engine\n")
	uci.stop("")
	//Log("engine stopped, checking move\n")
	//Log(fmt.Sprintf("current position: %s\n",uci.Engine.Position.String()))
	// make move if legal
	if move, err := uci.Engine.Position.UCIToMove(args[0]); err != nil {
		//Log("illegal move\n")
		return err
	} else {
		//Log("making move\n")
		uci.Engine.DoMove(move)
	}
	//Log("move made, check analyze\n")
	XBOARD_Check_Analyze()
	return nil
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// XBOARD_new : XBOARD new command
// -> uci *UCI : UCI
// <- error : error

func (uci *UCI) XBOARD_new() error {
	// reset board to the start position
	uci.SetVariant(VARIANT_CURRENT)
	XBOARD_Engine_Side = Black
	return nil
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// XBOARD_analyze : XBOARD analyze command
// -> uci *UCI : UCI
// <- error : error

func (uci *UCI) XBOARD_analyze() error {
	return nil
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// XBOARD_go : XBOARD go command
// switch engine to play side currently on move
// see: https://www.gnu.org/software/xboard/engine-intf.html
// -> uci *UCI : UCI
// <- error : error

func (uci *UCI) XBOARD_go() error {
	turn := uci.Engine.Position.SideToMove
	XBOARD_Engine_Side = turn
	return nil
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// XBOARD_playother : XBOARD playother command
// enabled by the feature command
// https://www.gnu.org/software/xboard/engine-intf.html
// "(This command is new in protocol version 2. It is not sent unless you enable it with the feature command.)
// Leave force mode and set the engine to play the color that is not on move.
// Associate the opponent's clock with the color that is on move, the engine's clock with the color that is not on move.
// Start the opponent's clock. If pondering is enabled, the engine should begin pondering.
// If the engine later receives a move, it should start thinking and eventually reply."
// -> uci *UCI : UCI
// <- error : error

func (uci *UCI) XBOARD_playother() error {
	turn := uci.Engine.Position.SideToMove
	XBOARD_Engine_Side = turn.Opposite()
	return nil
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// XBOARD_exit : XBOARD exit command
// -> uci *UCI : UCI
// <- error : error

func (uci *UCI) XBOARD_exit() error {
	// stop any ongoing analysis
	uci.stop("")
	return nil
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// XBOARD_force : XBOARD force command
// -> uci *UCI : UCI
// <- error : error

func (uci *UCI) XBOARD_force() error {
	uci.stop("")
	XBOARD_Engine_Side = NoColor
	return nil
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// XBOARD_undo : XBOARD undo command
// -> uci *UCI : UCI
// <- error : error

var XBOARD_undo_cnt = 0

func (uci *UCI) XBOARD_undo() error {
	if XBOARD_State == XBOARD_Analyzing {
		XBOARD_undo_cnt++
		switch XBOARD_undo_cnt {
			case 1 : return nil
			case 2 :
				XBOARD_undo_cnt = 0
				defer XBOARD_Check_Analyze()
		}
	}
	uci.stop("")
	// undo move
	uci.UndoMove(line)
	return nil
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// XBOARD_post : XBOARD post command
// -> uci *UCI : UCI
// <- error : error

func (uci *UCI) XBOARD_post() error {
	XBOARD_Post = true
	return nil
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// XBOARD_nopost : XBOARD nopost command
// -> uci *UCI : UCI
// <- error : error

func (uci *UCI) XBOARD_nopost() error {
	XBOARD_Post = false
	return nil
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// XBOARD_setboard : XBOARD setboard command
// -> uci *UCI : UCI
// <- error : error

func (uci *UCI) XBOARD_setboard() error {
	uci.stop("")
	fen := GetRest()
	//Log(fmt.Sprintf("setboard received fen %s\n",fen))
	pos, err := PositionFromFEN(fen)
	if err != nil {
		//Log("set from fen failed\n")
		return err
	}
	uci.Engine.SetPosition(pos)
	//Log(fmt.Sprintf("engine position set to %s\n",uci.Engine.Position.String()))
	return nil
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// XBOARD_level : XBOARD level command
// -> uci *UCI : UCI
// <- error : error

var reLevelTimeSeconds = regexp.MustCompile("([0-9]+):([0-9]+)")

func (uci *UCI) XBOARD_level() error {
	if numargs != 3 {
		return XBOARD_Error("wrong number of arguments for level",fmt.Sprintf("%d",numargs))
	}
	mvs, _ := strconv.Atoi(args[0])
	XBOARD_level_moves = mvs
	tmsmatch := reLevelTimeSeconds.FindStringSubmatch(args[1])
	if tmsmatch != nil {
		tms, _ := strconv.Atoi(tmsmatch[2])
		// time is given as seconds
		XBOARD_level_time = tms * 1000
	} else {
		tms, _ := strconv.Atoi(args[1])
		// time is given as minutes
		XBOARD_level_time = tms * 60 * 1000
	}
	incs, _ := strconv.Atoi(args[2])
	XBOARD_level_increment = incs * 1000
	return nil
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// XBOARD_time : XBOARD time command
// -> uci *UCI : UCI
// <- error : error

func (uci *UCI) XBOARD_time() error {
	if numargs != 1 {
		return XBOARD_Error("wrong number of arguments for time",fmt.Sprintf("%d",numargs))
	}
	// time given in centi seconds
	tmcs, _ := strconv.Atoi(args[0])
	// convert to milliseconds
	XBOARD_time = tmcs * 10

	if uci.timeControl == nil {
		return nil
	}

	if XBOARD_Engine_Side == White {
		uci.timeControl.WTime = time.Duration(XBOARD_time) * time.Millisecond
	} else {
		uci.timeControl.BTime = time.Duration(XBOARD_time) * time.Millisecond
	}

	return nil
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// XBOARD_time : XBOARD time command
// -> uci *UCI : UCI
// <- error : error

func (uci *UCI) XBOARD_otim() error {
	if numargs != 1 {
		return XBOARD_Error("wrong number of arguments for otim",fmt.Sprintf("%d",numargs))
	}
	// time given in centi seconds
	tmcs, _ := strconv.Atoi(args[0])
	// convert to milliseconds
	XBOARD_otim = tmcs * 10

	if uci.timeControl == nil {
		return nil
	}

	if XBOARD_Engine_Side == Black {
		uci.timeControl.WTime = time.Duration(XBOARD_otim) * time.Millisecond
	} else {
		uci.timeControl.BTime = time.Duration(XBOARD_otim) * time.Millisecond
	}

	return nil
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// XBOARD_Start_Thinking : start thinking
// can be caused by several XBOARD commands
// -> uci *UCI : UCI
// <- error : error

func (uci *UCI) XBOARD_Start_Thinking() error {
	predicted := uci.predicted == uci.Engine.Position.Zobrist()
	uci.timeControl = NewTimeControl(uci.Engine.Position, predicted)
	
	ponder := false

	// assume engine plays black
	wtime := XBOARD_otim
	btime := XBOARD_time

	if XBOARD_Engine_Side == White {
		// engine plays white
		wtime = XBOARD_time
		btime = XBOARD_otim
	}

	uci.timeControl.WTime = time.Duration(wtime) * time.Millisecond
	uci.timeControl.BTime = time.Duration(btime) * time.Millisecond

	// increment is same for both sides
	uci.timeControl.WInc = time.Duration(XBOARD_level_increment) * time.Millisecond
	uci.timeControl.BInc = time.Duration(XBOARD_level_increment) * time.Millisecond

	if XBOARD_level_moves > 0 {
		uci.timeControl.MovesToGo = XBOARD_level_moves
	} else {
		uci.timeControl.MovesToGo = 20
	}

	if ponder {
		// ponder was requested, so fill the channel
		// next write to uci.ponder will block
		uci.ponder <- struct{}{}
	}

	uci.timeControl.Start(ponder)
	uci.ready <- struct{}{}

	IgnoreMoves = []Move{}

	XBOARD_State = XBOARD_Thinking

	go uci.play()

	return nil
}

///////////////////////////////////////////////

// END XBOARD commands
///////////////////////////////////////////////

///////////////////////////////////////////////
// GetEngineName : determine engine name for given variant and protocol
// <- str string : engine name

func GetEngineName() string {
	return VARIANT_AND_PROTOCOL_TO_ENGINE_NAME[EngineNameIndex{variant: Variant, protocol: Protocol}]
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// Intro : introduction
// <- string : introduction

func Intro() string {
	return fmt.Sprintf("%s %s chess variant %s engine by %s\n",
		GetEngineName(),
		VARIANT_TO_NAME[Variant],
		PROTOCOL_TO_NAME[Protocol],
		Author)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// Printu : unbuffered write to stdout
// -> str string : string to be written

func Printu(str string) {
	os.Stdout.Write([]byte(str))
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// GetRest : get rest of arguments as a single string
// <- str string : rest of arguments joined by space

func GetRest() string {
	if (numargs-argptr)>0 {
		return strings.Join(args[argptr:]," ")
	} else {
		return ""
	}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// PrintBoard : prints the board
// -> uci *UCI : UCI

func (uci *UCI) PrintBoard() {
	uci.Engine.Position.PrintBoard()
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// newUCILogger : creates new uci logger
// <- *uciLogger : uci logger

func newUCILogger() *uciLogger {
	return &uciLogger{buf: &bytes.Buffer{}}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// BeginSearch : begin search
// -> ul *uciLogger : uci logger

func (ul *uciLogger) BeginSearch() {
	ul.start = time.Now()
	ul.buf.Reset()
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// EndSearch : end search
// -> ul *uciLogger : uci logger

func (ul *uciLogger) EndSearch() {
	ul.flush()
}

///////////////////////////////////////////////
// ReportPV : this function is called when the search finished a depth

func ReportPV() {
	// store latest available score
	if MultiPVList.HasScore() {
		// GetScore also does the sorting
		LastScore = MultiPVList.GetScore()
	} else {
		return
	}

	if DontPrintPV {
		// if pv should not be printed we are done
		return
	}

	if Protocol == PROTOCOL_UCI {
		for index := 0 ; index < len(MultiPVList) ; index ++ {
			info := fmt.Sprintf("info multipv %d %s", index+1, MultiPVList[index].InfoString)
			Printu(info)
			Log(info)
		}
	}

	if Protocol == PROTOCOL_XBOARD {
		// XBOARD does not support multipv mode, so we print the first pv
		Printu(MultiPVList[0].InfoString)
	}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// ReportInfoString : reports info string of a pv item that can be sent to the gui
// -> ul *uciLogger : uci logger
// -> item *MultiPVItem : multipv item
// <- string : info string

func (ul *uciLogger) ReportInfoString(item MultiPVItem) string {
	
	if Protocol == PROTOCOL_XBOARD {
		XBOARD_now := time.Now()
		XBOARD_elapsed := uint64(maxDuration(XBOARD_now.Sub(ul.start), time.Microsecond))
		XBOARD_millis := XBOARD_elapsed / uint64(time.Millisecond)
		XBOARD_centis := XBOARD_millis / 10
		XBOARD_score := item.Score
		if XBOARD_score > KnownWinScore {
			XBOARD_score = 100000 + (MateScore-XBOARD_score+1)/2
		} else if XBOARD_score < KnownLossScore {
			XBOARD_score = -100000 - (MatedScore-XBOARD_score)/2
		}
		buff := fmt.Sprintf("%d %d %d %d",
			item.Stats.Depth,
			XBOARD_score,
			XBOARD_centis,
			item.Stats.Nodes)
		for _, m := range item.Line {
			buff += fmt.Sprintf(" %v", m.UCI())
		}
		buff += "\n"
		return buff
	}

	if Protocol == PROTOCOL_UCI {
		buff := ""
		// write depth
		now := time.Now()
		buff += fmt.Sprintf("depth %d seldepth %d ", item.Stats.Depth, item.Stats.SelDepth)

		// write score
		if item.Score > KnownWinScore {
			buff += fmt.Sprintf("score mate %d ", (MateScore-item.Score+1)/2)
		} else if item.Score < KnownLossScore {
			buff += fmt.Sprintf("score mate %d ", (MatedScore-item.Score)/2)
		} else {
			buff += fmt.Sprintf("score cp %d ", item.Score)
		}

		// write stats
		elapsed := uint64(maxDuration(now.Sub(ul.start), time.Microsecond))
		nps := item.Stats.Nodes * uint64(time.Second) / elapsed
		millis := elapsed / uint64(time.Millisecond)
		buff += fmt.Sprintf("nodes %d time %d nps %d ", item.Stats.Nodes, millis, nps)

		// write principal variation
		buff += fmt.Sprintf("pv")
		for _, m := range item.Line {
			buff += fmt.Sprintf(" %v", m.UCI())
		}
		buff += "\n"

		return buff
	}

	// we should not get here
	return "?"
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// flush : flushes the buf to stdout
// -> ul *uciLogger : uci logger

func (ul *uciLogger) flush() {
	os.Stdout.Write(ul.buf.Bytes())
	os.Stdout.Sync()
	ul.buf.Reset()
}

///////////////////////////////////////////////
// maxDuration : returns maximum of a and b
// -> a time.Duration : duration a
// -> b time.Duration : duration b
// <- time.Duration : max(a,b)

func maxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}

///////////////////////////////////////////////
// NewUCI : creates new UCI
// <- *UCI : UCI

func NewUCI() *UCI {
	options := Options{}
	return &UCI{
		Engine:      NewEngine(nil, newUCILogger(), options),
		timeControl: nil,
		ready:       make(chan struct{}, 1),
		ponder:      make(chan struct{}, 1),
	}
}

///////////////////////////////////////////////
// Execute : execute UCI command
// -> uci *UCI : UCI
// -> line string : command line
// <- error : error

var reCmd = regexp.MustCompile(`^[[:word:]]+\b`)

var reF = regexp.MustCompile("^f ")

func (uci *UCI) Execute(line string) error {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	if TEST {
		// convenience command f for set from fen
		line=reF.ReplaceAllString(line,"position fen ")
	}

	cmd := reCmd.FindString(line)
	if cmd == "" {
		return fmt.Errorf("invalid command line")
	}

	// these commands do not expect the engine to be ready
	switch cmd {
	case "isready":
		return uci.isready(line)
	case "quit":
		return errQuit
	case "stop":
		return uci.stop(line)
	case "uci":
		return uci.uci(line)
	case "ponderhit":
		return uci.ponderhit(line)
	}

	// make sure that the engine is ready
	uci.ready <- struct{}{}
	<-uci.ready

	// these commands expect engine to be ready
	switch cmd {
	case "ucinewgame":
		return uci.ucinewgame(line)
	case "position":
		return uci.position(line)
	case "go":
		return uci.go_(line)
	case "setoption":
		return uci.setoption(line)
	default:
		return fmt.Errorf("unhandled command %s", cmd)
	}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// MakeSanMove : make san move
// -> uci *UCI : UCI
// -> line string : command line
// <- error : error

func (uci *UCI) MakeSanMove(line string) error {
	option := reMakeSanMove.FindStringSubmatch(line)
	if option == nil {
		XBOARD_Error("invalid make san move arguments",line)
		return errTestOk
	}
	move, err := uci.Engine.Position.SANToMove(option[1])
	if err != nil {
		XBOARD_Error("invalid move",option[1])
		return errTestOk
	}
	uci.Engine.DoMove(move)
	uci.PrintBoard()
	PrintBookPage()
	return nil
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// UndoMove : undo move
// -> uci *UCI : UCI
// -> line string : command line
// <- error : error

func (uci *UCI) UndoMove(line string) error {
	if uci.Engine.Position.GetNoStates()<2 {
		if Protocol == PROTOCOL_XBOARD {
			return nil
		}
		XBOARD_Error("no move to delete",line)
		return errTestOk
	}
	uci.Engine.UndoMove()
	if Protocol == PROTOCOL_UCI {
		uci.PrintBoard()
	}
	return nil
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// uci : execute uci command
// -> uci *UCI : UCI
// -> line string : command line
// <- error : error

func (uci *UCI) uci(line string) error {
	fmt.Printf("id name %s\n",GetEngineName())
	fmt.Printf("id author Alexandru Mosoi\n")
	fmt.Printf("\n")
	fmt.Printf("option name MultiPV type spin default 1 min 1 max 500\n")
	fmt.Printf("option name ClearHash type button\n")
	fmt.Printf("option name UseBook type button\n")
	if IS_Racing_Kings {
		for piece:=Knight ; piece<King ; piece++ {
			fmt.Printf("option name %s Value type spin default %d min 0 max 1000\n", 
					FigureToName[piece],RK_PIECE_VALUES[piece])
		}
		fmt.Printf("option name King Advance Value type spin default %d min 0 max 1000\n", KING_ADVANCE_VALUE)
	}
	fmt.Println("uciok")
	return nil
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// isready : isready command
// -> uci *UCI : UCI
// -> line string : command line
// <- error : error

func (uci *UCI) isready(line string) error {
	uci.ready <- struct{}{}
	<-uci.ready
	fmt.Println("readyok")
	return nil
}

///////////////////////////////////////////////
// -> ucinewgame : ucinewgame command
// -> uci *UCI : UCI
// -> line string : command line
// <- error : error

func (uci *UCI) ucinewgame(line string) error {
	// clear the hash at the beginning of each game
	GlobalHashTable.Clear()
	return nil
}

///////////////////////////////////////////////
//  position : position command
// -> uci *UCI : UCI
// -> line string : command line
// <- error : error

func (uci *UCI) position(line string) error {
	args := strings.Fields(line)[1:]
	if len(args) == 0 {
		return fmt.Errorf("expected argument for 'position'")
	}

	var pos *Position

	i := 0
	var err error
	switch args[i] {
	case "startpos":
		uci.SetVariant(VARIANT_CURRENT)
		i++
	case "fen":
		pos, err = PositionFromFEN(strings.Join(args[1:7], " "))
		if err != nil {
			return err
		}
		uci.Engine.SetPosition(pos)
		i += 7
	default:
		err = fmt.Errorf("unknown position command: %s", args[0])
		return err
	}

	if i < len(args) {
		if args[i] != "moves" {
			return fmt.Errorf("expected 'moves', got '%s'", args[1])
		}
		for _, m := range args[i+1:] {
			if move, err := uci.Engine.Position.UCIToMove(m); err != nil {
				return err
			} else {
				uci.Engine.DoMove(move)
			}
		}
	}

	return nil
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// GetSimpleBookMove : gets the best move for position from simple book
// -> pos *Position : position
// <- string : algeb
// <- bool : true if found

func (pos *Position) GetSimpleBookMove() (string, bool) {
	algeb, found := SimpleBook[pos.ZobristStr()]
	return algeb, found
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// go_ : go command
// -> uci *UCI : UCI
// -> line string : command line
// <- error : error

func (uci *UCI) go_(line string) error {
	if UseBook {
		pos := uci.Engine.Position
		algeb, found := pos.GetSimpleBookMove()
		if found && UseBook {
			_ , err := pos.UCIToMove(algeb)
			if err == nil {
				Printu(fmt.Sprintf("info depth 0 time 0 score cp 0 pv %s\nbestmove %s\n", algeb, algeb))
				return nil
			}
		}
	}
	// TODO: handle panic for `go depth`
	predicted := uci.predicted == uci.Engine.Position.Zobrist()
	uci.timeControl = NewTimeControl(uci.Engine.Position, predicted)
	uci.timeControl.MovesToGo = 30 // in case there is not time refresh
	ponder := false

	args := strings.Fields(line)[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "ponder":
			ponder = true
		case "infinite":
			uci.timeControl = NewTimeControl(uci.Engine.Position, false)
		case "wtime":
			i++
			t, _ := strconv.Atoi(args[i])
			uci.timeControl.WTime = time.Duration(t) * time.Millisecond
		case "winc":
			i++
			t, _ := strconv.Atoi(args[i])
			uci.timeControl.WInc = time.Duration(t) * time.Millisecond
		case "btime":
			i++
			t, _ := strconv.Atoi(args[i])
			uci.timeControl.BTime = time.Duration(t) * time.Millisecond
		case "binc":
			i++
			t, _ := strconv.Atoi(args[i])
			uci.timeControl.BInc = time.Duration(t) * time.Millisecond
		case "movestogo":
			i++
			t, _ := strconv.Atoi(args[i])
			uci.timeControl.MovesToGo = t
		case "movetime":
			i++
			t, _ := strconv.Atoi(args[i])
			uci.timeControl.WTime = time.Duration(t) * time.Millisecond
			uci.timeControl.WInc = 0
			uci.timeControl.BTime = time.Duration(t) * time.Millisecond
			uci.timeControl.BInc = 0
			uci.timeControl.MovesToGo = 1
		case "depth":
			i++
			d, _ := strconv.Atoi(args[i])
			uci.timeControl.Depth = int32(d)
		}
	}

	if ponder {
		// ponder was requested, so fill the channel
		// next write to uci.ponder will block
		uci.ponder <- struct{}{}
	}

	uci.timeControl.Start(ponder)
	uci.ready <- struct{}{}
	go uci.play()
	return nil
}

///////////////////////////////////////////////
// ponderhit : ponderhit command
// -> uci *UCI : UCI
// -> line string : command line
// <- error : error

func (uci *UCI) ponderhit(line string) error {
	uci.timeControl.PonderHit()
	<-uci.ponder
	return nil
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// stop : stop command
// -> uci *UCI : UCI
// -> line string : command line
// <- error : error

func (uci *UCI) stop(line string) error {
	// stop the timer if not already stopped
	if uci.timeControl != nil {
		uci.timeControl.Stop()
	}
	// no longer pondering
	select {
	case <-uci.ponder:
	default:
	}
	// waits until the engine becomes ready
	uci.ready <- struct{}{}
	<-uci.ready

	return nil
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// play : starts the engine
// should run in its own separate goroutine
// -> uci *UCI : UCI
// -> ignoremoves []Move : list of moves that should be ignored

func (uci *UCI) play() {

	if UseBook && ( Protocol == PROTOCOL_XBOARD ) && ( XBOARD_State != XBOARD_Analyzing ) {
		pos := uci.Engine.Position
		algeb, found := pos.GetSimpleBookMove()
		if found {
			move , err := pos.UCIToMove(algeb)
			if err == nil {
				uci.ponder <- struct{}{}
				<-uci.ponder

				Printu(fmt.Sprintf("move %s\n", algeb))
				uci.Engine.DoMove(move)
				XBOARD_State = XBOARD_Pondering

				<-uci.ready
				AddMoveChan <- 0
				return
			}
		}
	}

	moves := uci.Engine.Play(uci.timeControl, IgnoreMoves)

	if MultiPV > 1 {
		if MultiPVList.HasScore() {
			moves = MultiPVList[0].Line
		}
	}

	if len(moves) >= 2 {
		uci.Engine.Position.DoMove(moves[0])
		uci.Engine.Position.DoMove(moves[1])
		uci.predicted = uci.Engine.Position.Zobrist()
		uci.Engine.Position.UndoMove()
		uci.Engine.Position.UndoMove()
	} else {
		uci.predicted = uci.Engine.Position.Zobrist()
	}

	// if pondering was requested it will block because the channel is full
	uci.ponder <- struct{}{}
	<-uci.ponder

	IgnoreMoves = []Move{}

	if Protocol == PROTOCOL_UCI {
		if !DontPrintPV {
			if len(moves) == 0 {
				fmt.Printf("bestmove (none)\n")
			} else if len(moves) == 1 {
				fmt.Printf("bestmove %v\n", moves[0].UCI())
			} else {
				fmt.Printf("bestmove %v ponder %v\n", moves[0].UCI(), moves[1].UCI())
			}
		}

		if len(moves) > 0 {
			algeb := moves[0].UCI()
			if StoreScores {
				depth := int(uci.Engine.Stats.Depth)
				if depth >= StoreMinDepth {
					mentry , found := uci.Engine.Position.GetMoveEntry(algeb)
					ok := true
					if found {
						if depth < mentry.Depth {
							// if depth is lower than that of stored move
							// the score is only stored if book version is higher
							ok = BookVersion > mentry.BookVersion
						}
					}
					if ok {
						umentry := BookMoveEntry{
							Algeb : algeb,
							Score : int(LastScore),
							Depth : depth,
							BookVersion : BookVersion,
							HasEval : false,
							Eval : 0,
						}
						uci.Engine.Position.StoreMoveEntry(algeb, umentry)
					}
				}
			}
			if MakeAnalyzedMove {
				uci.Engine.DoMove(moves[0])
				uci.PrintBoard()
				MakeAnalyzedMove = false
			}
		}
	}

	if Protocol == PROTOCOL_XBOARD {
		if len(moves) > 0 {
			if XBOARD_State != XBOARD_Analyzing {
				uci.Engine.DoMove(moves[0])
				XBOARD_State = XBOARD_Pondering
			}
			if XBOARD_do_hint {
				Printu(fmt.Sprintf("Hint: %s\n", moves[0].UCI()))
				XBOARD_do_hint = false
			} else {
				Printu(fmt.Sprintf("move %s\n", moves[0].UCI()))
			}
		}
	}

	// marks the engine as ready
	// if the engine is made ready before best move is shown
	// then sometimes (at very high rate of commands position / go)
	// there is a race info / bestmove lines are intermixed wrongly
	// this confuses the tuner, at least
	<-uci.ready

	AddMoveChan <- 0
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// setoption : setoption command
// -> uci *UCI : UCI
// -> line string : command line
// <- error : error

var reOption = regexp.MustCompile(`^setoption\s+name\s+(.+?)(\s+value\s*(.*))?$`)

var reRkSetPieceValue = regexp.MustCompile("^([^\\s]+)\\s+Value$")

var reMakeSanMove = regexp.MustCompile(`^m\s+([^\s]+)$`)

func (uci *UCI) setoption(line string) error {
	option := reOption.FindStringSubmatch(line)
	if option == nil {
		return fmt.Errorf("invalid setoption arguments")
	}

	// handle buttons which don't have a value
	if len(option) < 1 {
		return fmt.Errorf("missing setoption name")
	}
	switch option[1] {
	case "ClearHash":
		GlobalHashTable.Clear()
		return nil
	case "UseBook":
		UseBook = true
		return nil
	}

	// handle remaining values.
	if len(option) < 3 {
		return fmt.Errorf("missing setoption value")
	}

	///////////////////////////////////////////////////
	// NEW
	if IS_Racing_Kings {
		setPieceValue := reRkSetPieceValue.FindStringSubmatch(option[1])
		if setPieceValue != nil {
			pieceValue , err := strconv.ParseInt(option[3], 10, 32)
			if err != nil {
				return fmt.Errorf("wrong piece value")
			}
			RK_PIECE_VALUES[FigureNameToFigure(setPieceValue[1])]=int32(pieceValue)
			return nil
		}
		switch option[1] {
		case "King Advance Value" :
			kingAdvanceValue , err := strconv.ParseInt(option[3], 10, 32)
			if err != nil {
				return fmt.Errorf("wrong king advance value")
			}
			KING_ADVANCE_VALUE = int32(kingAdvanceValue)
			return nil
		}
	}
	// END NEW
	///////////////////////////////////////////////////

	switch option[1] {
	case "MultiPV":
	if multipv, err := strconv.ParseInt(option[3], 10, 64); err != nil {
		return err
	} else {
		MultiPV = int(multipv)
	}
	return nil
	case "UCI_AnalyseMode":
		if mode, err := strconv.ParseBool(option[3]); err != nil {
			return err
		} else {
			uci.Engine.Options.AnalyseMode = mode
		}
		return nil
	case "Hash":
		if hashSizeMB, err := strconv.ParseInt(option[3], 10, 64); err != nil {
			return err
		} else {
			GlobalHashTable = NewHashTable(int(hashSizeMB))
		}
		return nil
	default:
		return fmt.Errorf("unhandled option %s", option[1])
	}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// SetVariant : set variant
// -> uci *UCI : UCI
// <- error : error

func (uci *UCI) SetVariant(setVariant int) error {
	switch Protocol {
		case PROTOCOL_UCI: log.SetPrefix("info string ")
		case PROTOCOL_XBOARD: log.SetPrefix("Error ")
	}
	uci.Engine.SetVariant(setVariant)
	return nil
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// AnnotateMove : annotate move
// <- error : error

func AnnotateMove() error {
	san := args[0]
	annot, _ := strconv.Atoi(args[1])

	pos := uci.Engine.Position

	move, err := pos.SANToMove(san)

	if err == nil {
		algeb := move.UCI()
		mentry, found := pos.GetMoveEntry(algeb)
		if found {
			mentry.Int1 = annot
		} else {
			mentry = BookMoveEntry{
				Algeb : algeb,
				Score : 0,
				Depth : 0,
				BookVersion : BookVersion,
				HasEval : false,
				Eval : 0,
				Nodes : 0,
				Int1 : annot,
			}
		}
		pos.StoreMoveEntry(algeb, mentry)
	} else {
		return XBOARD_Error("illegal annotation move",san)
	}
	return nil
}

///////////////////////////////////////////////