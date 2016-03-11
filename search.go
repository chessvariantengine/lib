//////////////////////////////////////////////////////
// search.go
// implements the search
// zurichess sources: engine.go, material.go, move_ordering.go, pv.go, score.go, cache.go
//////////////////////////////////////////////////////

package lib

// imports

import(
	"time"
	"sync"
	"fmt"
)

///////////////////////////////////////////////
// definitions

var RK_PIECE_VALUES = []int32{
	0,
	0,
	300,
	325,
	500,
	700,
}

var KING_ADVANCE_VALUE int32 = 250
var KNIGHT_ADVANCE_VALUE int32 = 5

const (
	KnownWinScore  int32 = 25000000       // KnownWinScore is strictly greater than all evaluation scores (mate not included).
	KnownLossScore int32 = -KnownWinScore // KnownLossScore is strictly smaller than all evaluation scores (mated not included).
	MateScore      int32 = 30000000       // MateScore - N is mate in N plies.
	MatedScore     int32 = -MateScore     // MatedScore + N is mated in N plies.
	InfinityScore  int32 = 32000000       // InfinityScore is possible score. -InfinityScore is the minimum possible score.
)

// cache implements a fixed size cache
type cache struct {
	table []cacheEntry
	hash  func(*Position, Color) uint64
	comp  func(*Position, Color) Eval
}

// cacheEntry is a cache entry
type cacheEntry struct {
	lock uint64
	eval Eval
}

var (
	// weights stores all evaluation parameters under one array for easy handling
	// Zurichess' evaluation is a very simple neural network with no hidden layers,
	// and one output node y = W_m * x * (1-p) + W_e * x * p where W_m are
	// middle game weights, W_e are endgame weights, x is input, p is phase between
	// middle game and end game, and y is the score
	// the network has |x| = len(Weights) inputs corresponding to features
	// extracted from the position. These features are symmetrical wrt to colors
	// the network is trained using the Texel's Tuning Method
	// https://chessprogramming.wikispaces.com/Texel%27s+Tuning+Method
	Weights = [94]Score{
		{M: 1034, E: 5770}, {M: 5363, E: 9844}, {M: 39652, E: 54153}, {M: 42277, E: 58849}, {M: 57185, E: 103947},
		{M: 140637, E: 189061}, {M: 4799, E: 7873}, {M: 9625, E: 9558}, {M: 950, E: 2925}, {M: 1112, E: 1908},
		{M: 806, E: 1167}, {M: 732, E: 824}, {M: 168, E: 1149}, {M: -879, E: -359}, {M: 3495, E: 7396},
		{M: 2193, E: 7557}, {M: 1909, E: 7559}, {M: 3903, E: 3354}, {M: 3372, E: 6143}, {M: 7773, E: 5680},
		{M: 6441, E: 4512}, {M: 2974, E: 2896}, {M: 3912, E: 6372}, {M: 2689, E: 6273}, {M: 3266, E: 4799},
		{M: 3581, E: 4578}, {M: 4765, E: 6213}, {M: 5273, E: 5606}, {M: 5775, E: 4043}, {M: 3817, E: 4274},
		{M: 3708, E: 8782}, {M: 2391, E: 7627}, {M: 5072, E: 4626}, {M: 6109, E: 3746}, {M: 5668, E: 5198},
		{M: 3913, E: 5131}, {M: 2830, E: 5977}, {M: 2266, E: 5967}, {M: 3516, E: 10438}, {M: 3637, E: 8738},
		{M: 4903, E: 5959}, {M: 5655, E: 3593}, {M: 5049, E: 5557}, {M: 5400, E: 4573}, {M: 3630, E: 7749},
		{M: 2604, E: 7455}, {M: 5493, E: 12869}, {M: 5021, E: 10574}, {M: 8042, E: 6544}, {M: 10390, E: -1256},
		{M: 11098, E: -2344}, {M: 12808, E: 4315}, {M: 8494, E: 9675}, {M: 7990, E: 9444}, {M: 13836, E: 17481},
		{M: 12537, E: 16982}, {M: 11116, E: 10810}, {M: 15238, E: 3620}, {M: 10331, E: 2338}, {M: 6943, E: 8458},
		{M: -835, E: 14771}, {M: -1276, E: 18329}, {M: 7371, E: 5198}, {M: 256, E: 1926}, {M: -53, E: 2965},
		{M: -254, E: 6546}, {M: 2463, E: 10465}, {M: 5507, E: 19296}, {M: 11056, E: 20099}, {M: 8034, E: 5202},
		{M: 4857, E: -3126}, {M: 3065, E: 3432}, {M: -137, E: 6127}, {M: -2620, E: 8577}, {M: -9391, E: 12415},
		{M: -3313, E: 12592}, {M: 7738, E: 8987}, {M: 18783, E: -215}, {M: -526, E: 755}, {M: 6310, E: 5426},
		{M: 5263, E: 7710}, {M: -2482, E: 10646}, {M: 2399, E: 8982}, {M: -607, E: 9555}, {M: 7854, E: 5619},
		{M: 5386, E: 402}, {M: 1228, E: 866}, {M: -991, E: 178}, {M: -1070, E: -1129}, {M: 2183, E: 362},
		{M: -2259, E: -681}, {M: 3854, E: 9184}, {M: 4472, E: 890}, {M: 1300, E: 1524},
	}

	// named chunks of Weights
	wFigure             [FigureArraySize]Score
	wMobility           [FigureArraySize]Score
	wPawn               [48]Score
	wPassedPawn         [8]Score
	wKingRank           [8]Score
	wKingFile           [8]Score
	wConnectedPawn      Score
	wDoublePawn         Score
	wIsolatedPawn       Score
	wPawnThreat         Score
	wKingShelter        Score
	wBishopPair         Score
	wRookOnOpenFile     Score
	wRookOnHalfOpenFile Score

	// evaluation caches
	pawnsAndShelterCache *cache
)

const (
	CheckDepthExtension    int32 = 1 // how much to extend search in case of checks
	NullMoveDepthLimit     int32 = 1 // disable null-move below this limit
	NullMoveDepthReduction int32 = 1 // default null-move depth reduction, can reduce more in some situations
	PVSDepthLimit          int32 = 0 // do not do PVS below and including this limit
	LMRDepthLimit          int32 = 3 // do not do LMR below and including this limit
	FutilityDepthLimit     int32 = 3 // maximum depth to do futility pruning

	initialAspirationWindow = 21  // ~a quarter of a pawn
	futilityMargin          = 150 // ~one and a halfpawn
	checkpointStep          = 10000
)

var (
	// scoreMultiplier is used to compute the score from side
	// to move POV from given the score from white POV
	scoreMultiplier = [ColorArraySize]int32{0, -1, 1}
)

// Options keeps engine's options
type Options struct {
	AnalyseMode bool // true to display info strings
}

// stats stores some basic stats of the search
// statistics are reset every iteration of the iterative deepening search
type Stats struct {
	CacheHit  uint64 // number of times the position was found transposition table
	CacheMiss uint64 // number of times the position was not found in the transposition table
	Nodes     uint64 // number of nodes searched
	Depth     int32  // depth search
	SelDepth  int32  // maximum depth reached on PV (doesn't include the hash moves)
}

// CacheHitRatio returns the ration of hits over total number of lookups
func (s *Stats) CacheHitRatio() float32 {
	return float32(s.CacheHit) / float32(s.CacheHit+s.CacheMiss)
}

// Logger logs search progress
type Logger interface {
	// BeginSearch signals a new search is started
	BeginSearch()
	// EndSearch signals end of search
	EndSearch()
	// PrintPV logs the principal variation after
	// iterative deepening completed one depth
	PrintPV(stats Stats, score int32, pv []Move)
}

// NulLogger is a logger that does nothing.
type NulLogger struct {
}

func (nl *NulLogger) BeginSearch() {
}

func (nl *NulLogger) EndSearch() {
}

func (nl *NulLogger) PrintPV(stats Stats, score int32, pv []Move) {
}

// historyEntry keeps counts of how well move performed in the past
type historyEntry struct {
	counter [2]int
	move    Move
}

// historyTable is a hash table that contains history of moves
// old moves are automatically evicted when new moves are inserted
// so this cache is approx. LRU
type historyTable []historyEntry

// movesStack is a stack of moves
type moveStack struct {
	moves []Move  // list of moves
	order []int16 // weight of each move for comparison

	kind   int     // violent or all
	state  int     // current generation state
	hash   Move    // hash move
	killer [4]Move // killer moves
}

// stack is a stack of plies (movesStack)
type stack struct {
	position *Position
	moves    []moveStack
}

// TODO: Unexport pvEntry fields.
type pvEntry struct {
	// lock is used to handled hash conflicts
	// normally set to position's Zobrist key
	lock uint64
	// when was the move added
	birth uint32
	// move on pricipal variation for this position
	move Move
}

// pvTable is like hash table, but only to keep principal variation
// the additional table to store the PV was suggested by Robert Hyatt, see
// * http://www.talkchess.com/forum/viewtopic.php?topic_view=threads&p=369163&t=35982
// * http://www.talkchess.com/forum/viewtopic.php?t=36099
// during alpha-beta search entries that are on principal variation,
// are exact nodes, i.e. their score lies exactly between alpha and beta

type pvTable struct {
	table []pvEntry
	timer uint32
}

// atomicFlag is an atomic bool that can only be set
type atomicFlag struct {
	lock sync.Mutex
	flag bool
}

// TimeControl is a time control that tries to split the
// remaining time over MovesToGo
type TimeControl struct {
	WTime, WInc time.Duration // time and increment for white
	BTime, BInc time.Duration // time and increment for black
	Depth       int32         // maximum depth search (including)
	MovesToGo   int           // number of remaining moves

	sideToMove Color
	time, inc  time.Duration // time and increment for us
	limit      time.Duration

	predicted bool       // true if this move was predicted
	branch    int        // branching factor
	currDepth int32      // current depth searched
	stopped   atomicFlag // true to stop the search
	ponderhit atomicFlag // true if ponder was successful

	searchTime     time.Duration // alocated time for this move
	searchDeadline time.Time     // don't go to the next depth after this deadline
	stopDeadline   time.Time     // abort search after this deadline
}

// Engine implements the logic to search the best move for a position
type Engine struct {
	Options  Options   // engine options
	Log      Logger    // logger
	Stats    Stats     // search statistics
	Position *Position // current Position

	rootPly int          // position's ply at the start of the search
	stack   stack        // stack of moves
	pvTable pvTable      // principal variation table
	history historyTable // keeps history of moves

	timeControl *TimeControl
	stopped     bool
	checkpoint  uint64
}

const (
	pvTableSize = 1 << 13
	pvTableMask = pvTableSize - 1
)

const disableCache = false

// Score represents a pair of mid and end game scores.
type Score struct {
	M, E int32 // mid game, end game
}

// Eval is a sum of scores.
type Eval struct {
	M, E int32 // mid game, end game
}

// end definitions
///////////////////////////////////////////////

///////////////////////////////////////////////
// newHistoryTable : creates new history table
// <- historyTable : created table

func newHistoryTable() historyTable {
	return make([]historyEntry, 1024)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// historyHash : hashes the move and returns an index into the history table
// -> m Move : move
// <- uint32 : index

func historyHash(m Move) uint32 {
	// this is a murmur inspired hash so upper bits are better
	// mixed than the lower bits, the hash multiplier was chosen
	// to minimize the number of misses
	h := uint32(m) * 438650727
	return (h + (h << 17)) >> 22
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// get : returns counters for m, i.e. pair of (bad, good)
// TODO: consider returning only if the move is good or bad
// -> ht historyTable : history table
// -> m Move : move
// <- int : bad count
// <- int : good count

func (ht historyTable) get(m Move) (int, int) {
	h := historyHash(m)
	if ht[h].move != m {
		return 0, 0
	}
	return ht[h].counter[0], ht[h].counter[1]
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// inc : increments the counters for m
// evicts an old move if necessary
// counters start from 1 so probability is correctly estimated, TODO: insert reference
// -> ht historyTable : history table
// -> m Move : move
// -> good bool : good

func (ht historyTable) inc(m Move, good bool) {
	h := historyHash(m)
	if ht[h].move != m {
		ht[h] = historyEntry{
			counter: [2]int{1, 1},
			move:    m,
		}
	}
	if good {
		ht[h].counter[1]++
	} else {
		ht[h].counter[0]++
	}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// newPvTable : returns a new pvTable
// <- pvTable : pv table

func newPvTable() pvTable {
	return pvTable{
		table: make([]pvEntry, pvTableSize),
		timer: 0,
	}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// SetPosition : sets current position
// if pos is nil, the starting position is set
// -> eng *Engine : engine
// -> pos *Position : position

func (eng *Engine) SetPosition(pos *Position) {
	if pos != nil {
		eng.Position = pos
	} else {
		eng.Position, _ = PositionFromFEN(FENStartPos)
	}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// NewEngine : creates a new engine to search for pos
// if pos is nil then the start position is used
// -> pos *Position : position
// -> log Logger : logger
// -> options Options : options
// <- *Engine : engine

func NewEngine(pos *Position, log Logger, options Options) *Engine {
	if log == nil {
		log = &NulLogger{}
	}
	eng := &Engine{
		Options: options,
		Log:     log,
		pvTable: newPvTable(),
		history: newHistoryTable(),
	}
	eng.SetPosition(pos)
	return eng
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// FigureNameToFigure : figure name to figure
// -> figureString string : figure name as string
// <- Figure : figure

func FigureNameToFigure(figureString string) Figure {
	switch figureString {
		case "Pawn" : return Pawn
		case "Knight" : return Knight
		case "Bishop" : return Bishop
		case "Rook" : return Rook
		case "Queen" : return Queen
		case "King" : return King
	}
	return NoFigure
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// PrintPieceValues : prints piece values

func PrintPieceValues() {
	if Variant == VARIANT_Racing_Kings {
		for i:=Knight; i<King ; i++ {
			fmt.Printf("%s %d\n",FigureToName[i],RK_PIECE_VALUES[i])
		}
		fmt.Printf("King Advance %d\n",KING_ADVANCE_VALUE)
	}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// SetVariant : sets Variant and resets the engine to its starting position
// -> eng *Engine : engine
// -> setVariant int : variant

func (eng *Engine) SetVariant(setVariant int) {
	if(setVariant<0) {
		setVariant=Variant
	}
	Variant=setVariant
	pos, _ := PositionFromFEN(START_FENS[Variant])
	eng.SetPosition(pos)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// DoMove : executes a move
// -> eng *Engine : engine
// -> move Move : move

func (eng *Engine) DoMove(move Move) {
	eng.Position.DoMove(move)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// UndoMove : undoes the last move
// -> eng *Engine : engine
// -> move Move : move

func (eng *Engine) UndoMove() {
	eng.Position.UndoMove()
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// EvaluateSideRk : evaluate side for Racing Kings
// -> pos *Position : position
// -> side Color : side
// <- int32 : eval

func EvaluateSideRk(pos *Position, side Color) int32 {
	var val int32 = 0
	// piece values
	for piece := Knight ; piece < King ; piece++ {
		num := pos.ByPiece(side, piece).Count()
		val += num * RK_PIECE_VALUES[piece]
	}
	// king advance value
	val += int32(pos.ByPiece(side, King).AsSquare().Rank())*KING_ADVANCE_VALUE
	// knight advance value
	for bb := pos.ByPiece(side, Knight); bb > 0; {
		sq := bb.Pop()
		val += int32(sq.Rank())*KNIGHT_ADVANCE_VALUE
	}
	return val
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// Feed : eval feed
// -> e *Eval : eval
// -> phase int32 : phase
// <- int32 : eval

func (e *Eval) Feed(phase int32) int32 {
	return (e.M*(256-phase) + e.E*phase) / 256
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// Merge : merge eval
// -> e *Eval : eval
// -> o Eval : eval to be merged

func (e *Eval) Merge(o Eval) {
	e.M += o.M
	e.E += o.E
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// Add : add score to eval
// -> e *Eval : eval
// -> s Score : score to be added

func (e *Eval) Add(s Score) {
	e.M += s.M
	e.E += s.E
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// Add : add score to eval n times
// -> e *Eval : eval
// -> s Score : score to be added
// -> n int32 : times score to be added

func (e *Eval) AddN(s Score, n int32) {
	e.M += s.M * n
	e.E += s.E * n
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// Neg : negate eval
// -> e *Eval : eval

func (e *Eval) Neg() {
	e.M = -e.M
	e.E = -e.E
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// initWeights : init weights

func initWeights() {
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// newCache : creates a new cache of size 1<<bits
// -> bits uint : bits
// -> hash func(*Position, Color) uint64 : hash func
// -> comp func(*Position, Color) Eval : comp func
// <- *cache : cache

func newCache(bits uint, hash func(*Position, Color) uint64, comp func(*Position, Color) Eval) *cache {
	return &cache{
		table: make([]cacheEntry, 1<<bits),
		hash:  hash,
		comp:  comp,
	}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// put : puts a new entry in the cache
// -> c *cache : cache
// -> lock uint64 : lock
// -> eval Eval : eval

func (c *cache) put(lock uint64, eval Eval) {
	indx := lock & uint64(len(c.table)-1)
	c.table[indx] = cacheEntry{lock: lock, eval: eval}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// get : gets an entry from the cache
// -> c *cache : cache
// -> lock uint64 : lock
// <- Eval : eval
// <- bool : ok

func (c *cache) get(lock uint64) (Eval, bool) {
	indx := lock & uint64(len(c.table)-1)
	return c.table[indx].eval, c.table[indx].lock == lock
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// load : evaluates position, using the cache if possible
// -> c *cache : cache
// -> pos *Position : position
// -> us Color : side
// <- Eval : eval

func (c *cache) load(pos *Position, us Color) Eval {
	if disableCache {
		return c.comp(pos, us)
	}
	h := c.hash(pos, us)
	if e, ok := c.get(h); ok {
		return e
	}
	e := c.comp(pos, us)
	c.put(h, e)
	return e
}


///////////////////////////////////////////////

///////////////////////////////////////////////
// ScaleToCentiPawn : scales the score returned by Evaluate
// such that one pawn ~= 100
// -> score int32 : score
// <- int32 : centi pawn score

func ScaleToCentiPawn(score int32) int32 {
	return (score + 64) / 128
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// evaluateSide : evaluates position for a single side
// -> pos *Position : position
// -> us Color : us
// -> eval *Eval : eval

func evaluateSide(pos *Position, us Color, eval *Eval) {
	eval.Merge(pawnsAndShelterCache.load(pos, us))
	all := pos.ByColor[White] | pos.ByColor[Black]
	them := us.Opposite()

	// Pawn
	mobility := Forward(us, pos.ByPiece(us, Pawn)) &^ all
	eval.AddN(wMobility[Pawn], mobility.Count())
	mobility = pos.PawnThreats(us) & pos.ByColor[us.Opposite()]
	eval.AddN(wPawnThreat, mobility.Count())

	// Knight
	excl := pos.ByPiece(us, Pawn) | pos.PawnThreats(them)
	for bb := pos.ByPiece(us, Knight); bb > 0; {
		sq := bb.Pop()
		eval.Add(wFigure[Knight])
		mobility := KnightMobility(sq) &^ excl
		eval.AddN(wMobility[Knight], mobility.Count())
	}
	// Bishop
	numBishops := int32(0)
	for bb := pos.ByPiece(us, Bishop); bb > 0; {
		sq := bb.Pop()
		eval.Add(wFigure[Bishop])
		mobility := BishopMobility(sq, all) &^ excl
		eval.AddN(wMobility[Bishop], mobility.Count())
		numBishops++
	}
	eval.AddN(wBishopPair, numBishops/2)

	// Rook
	for bb := pos.ByPiece(us, Rook); bb > 0; {
		sq := bb.Pop()
		eval.Add(wFigure[Rook])
		mobility := RookMobility(sq, all) &^ excl
		eval.AddN(wMobility[Rook], mobility.Count())

		// Evaluate rook on open and semi open files.
		// https://chessprogramming.wikispaces.com/Rook+on+Open+File
		f := FileBb(sq.File())
		if pos.ByPiece(us, Pawn)&f == 0 {
			if pos.ByPiece(them, Pawn)&f == 0 {
				eval.Add(wRookOnOpenFile)
			} else {
				eval.Add(wRookOnHalfOpenFile)
			}
		}
	}
	// Queen
	for bb := pos.ByPiece(us, Queen); bb > 0; {
		sq := bb.Pop()
		eval.Add(wFigure[Queen])
		mobility := QueenMobility(sq, all) &^ excl
		eval.AddN(wMobility[Queen], mobility.Count())
	}

	// King, each side has one.
	{
		sq := pos.ByPiece(us, King).AsSquare()
		mobility := KingMobility(sq) &^ excl
		eval.AddN(wMobility[King], mobility.Count())
	}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// evaluatePosition : evalues position
// -> pos *Position : position
// <- Eval : eval

func EvaluatePosition(pos *Position) Eval {
	var eval Eval
	evaluateSide(pos, Black, &eval)
	eval.Neg()
	evaluateSide(pos, White, &eval)
	return eval
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// Phase : computes the progress of the game
// 0 is opening, 256 is late end game
// -> pos *Position : position
// <- int32 : phase

func Phase(pos *Position) int32 {
	total := int32(4*1 + 4*1 + 4*2 + 2*4)
	curr := total
	curr -= pos.ByFigure[Knight].Count() * 1
	curr -= pos.ByFigure[Bishop].Count() * 1
	curr -= pos.ByFigure[Rook].Count() * 2
	curr -= pos.ByFigure[Queen].Count() * 4
	return (curr*256 + total/2) / total
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// Evaluate : evaluates position from White's POV
// -> pos *Position : position
// <- int32 : eval

func Evaluate(pos *Position) int32 {
	///////////////////////////////////////////////////
	// NEW
	if Variant == VARIANT_Racing_Kings {
		evalw := EvaluateSideRk(pos, White)
		evalb := EvaluateSideRk(pos, Black)

		eval := evalw - evalb

		score := eval*128

		return score
	}
	///////////////////////////////////////////////////
	eval := EvaluatePosition(pos)
	score := eval.Feed(Phase(pos))
	if KnownLossScore >= score || score >= KnownWinScore {
		panic(fmt.Sprintf("score %d should be between %d and %d",
			score, KnownLossScore, KnownWinScore))
	}
	return score
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// Score : evaluates current position from current player's POV
// -> eng *Engine : engine
// <- int32 : score

func (eng *Engine) Score() int32 {
	score := Evaluate(eng.Position)
	score = ScaleToCentiPawn(score)
	return scoreMultiplier[eng.Position.SideToMove] * score
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// ply : returns the ply from the beginning of the search
// -> eng *Engine : engine
// <- int32 : ply

func (eng *Engine) ply() int32 {
	return int32(eng.Position.Ply - eng.rootPly)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// InsufficientMaterial : returns true if the position is theoretical draw
// -> pos *Position : position
// <- bool : true if material insufficient

func (pos *Position) InsufficientMaterial() bool {
	///////////////////////////////////////////////////
	// NEW
	if Variant == VARIANT_Racing_Kings {
		if pos.IsOnBaseRank(White) && pos.IsOnBaseRank(Black) {
			// Both kings on base rank is draw.
			return true
		}
		// No other insufficient material condition for Racking Kings.
		return false
	}
	///////////////////////////////////////////////////

	// K vs K is draw
	noKings := (pos.ByColor[White] | pos.ByColor[Black]) &^ pos.ByFigure[King]
	if noKings == 0 {
		return true
	}
	// KN vs K is theoretical draw
	if noKings == pos.ByFigure[Knight] && pos.ByFigure[Knight].CountMax2() == 1 {
		return true
	}
	// KB* vs KB* is theoretical draw if all bishops are on the same square color
	if bishops := pos.ByFigure[Bishop]; noKings == bishops {
		if bishops&BbWhiteSquares == bishops ||
			bishops&BbBlackSquares == bishops {
			return true
		}
	}
	return false
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// FiftyMoveRule returns True if 50 moves (on each side) were made
// without any capture of pawn move
// if FiftyMoveRule returns true, the position is a draw
// -> pos *Position : position
// <- bool : true if position is draw by 50 moves rule

func (pos *Position) FiftyMoveRule() bool {
	return pos.curr.HalfmoveClock >= 100
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// ThreeFoldRepetition : returns whether current position was seen three times already
// returns minimum between 3 and the actual number of repetitions
// -> pos *Position : position
// <- int : repetition count

func (pos *Position) ThreeFoldRepetition() int {
	c, z := 0, pos.Zobrist()
	for i := 0; i < len(pos.states) && i <= pos.curr.HalfmoveClock; i += 2 {
		if pos.states[len(pos.states)-1-i].Zobrist == z {
			if c++; c == 3 {
				break
			}
		}
	}
	return c
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// endPosition : determines whether the current position is an end game
// returns score and a bool if the game has ended
// -> eng *Engine : engine
// <- int32 : score
// <- bool : true if game ended

func (eng *Engine) endPosition() (int32, bool) {
	pos := eng.Position // shortcut
	// Trivial cases when kings are missing.
	if pos.ByPiece(White, King) == 0 && pos.ByPiece(Black, King) == 0 {
		return 0, true
	}
	if pos.ByPiece(White, King) == 0 {
		return scoreMultiplier[pos.SideToMove] * (MatedScore + eng.ply()), true
	}
	if pos.ByPiece(Black, King) == 0 {
		return scoreMultiplier[pos.SideToMove] * (MateScore - eng.ply()), true
	}
	// Neither side cannot mate.
	if pos.InsufficientMaterial() {
		return 0, true
	}
	// Fifty full moves without a capture or a pawn move.
	if pos.FiftyMoveRule() {
		return 0, true
	}
	// Repetition is a draw.
	// At root we need to continue searching even if we saw two repetitions already,
	// however we can prune deeper search only at two repetitions.
	if r := pos.ThreeFoldRepetition(); eng.ply() > 0 && r >= 2 || r >= 3 {
		return 0, true
	}
	return 0, false
}

///////////////////////////////////////////////

///////////////////////////////////////////////
//  : 
// ->  : 
// <-  : 

///////////////////////////////////////////////