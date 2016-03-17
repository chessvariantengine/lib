//////////////////////////////////////////////////////
// search.go
// implements the search
// zurichess sources: engine.go, material.go, move_ordering.go, pv.go, score.go
// cache.go, hash_table.go, score.go, see.go, time_control.go
//////////////////////////////////////////////////////

package lib

// imports

import(
	"time"
	"sync"
	"fmt"
	"unsafe"
)

///////////////////////////////////////////////
// definitions

// racing kings piece values
var RK_PIECE_VALUES = []int32{
	0,
	0,
	300,
	325,
	500,
	700,
}

// king advance value for racing kings
var KING_ADVANCE_VALUE int32    = 250

// knight advance value for racing kings
var KNIGHT_ADVANCE_VALUE int32  = 5

// atomic pawn bonus
var ATOMIC_PAWN_BONUS           = 150

// atomic pawn bonus score
var ATOMIC_PAWN_BONUS_SCORE     = Score{ M: int32(ATOMIC_PAWN_BONUS*128) , E: int32(ATOMIC_PAWN_BONUS*128) }

// horde pawn scores
var HORDE_PAWN_SCORES [ColorArraySize]Score

// balance material inequality in horde
var HORDE_BALANCE_SCORE         = Score{ M: int32(-1400*128) , E: int32(-1400*128) }

// horde center bonus score
var HORDE_CENTER_BONUS          = Score{ M: int32(128) , E: int32(128) }

// horde center bonus weights
var HORDE_CENTER_BONUS_WEIGHTS  = [...]int32{ 0, 80, 120, 150, 150, 120, 80, 0 }

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

var (
	DefaultHashTableSizeMB = 64       // DefaultHashTableSizeMB is the default size in MB
	GlobalHashTable        *HashTable // GlobalHashTable is the global transposition table
)

// hashKing type
type hashKind uint8

const (
	noEntry    hashKind = iota // no entry
	exact                      // exact score is known
	failedLow                  // search failed low, upper bound
	failedHigh                 // search failed high, lower bound
)

// hashEntry is a value in the transposition table
type hashEntry struct {
	lock  uint32   // lock is used to handle hashing conflicts
	move  Move     // best move
	score int32    // score of the position, if mate, score is relative to current position
	depth int8     // remaining search depth
	kind  hashKind // type of hash
}

// HashTable is a transposition table
// engine uses this table to cache position scores so
// it doesn't have to research them again.
type HashTable struct {
	table []hashEntry // len(table) is a power of two and equals mask+1
	mask  uint32      // mask is used to determine the index in the table
}

const (
	// Move generation states.

	msHash          = iota // return hash move
	msGenViolent           // generate violent moves
	msReturnViolent        // return violent moves in order
	msGenKiller            // generate killer moves
	msReturnKiller         // return killer moves  in order
	msGenRest              // generate remaining moves
	msReturnRest           // return remaining moves in order
	msDone                 // all moves returned
)

var (
	// mvvlva values based on one pawn = 10.
	mvvlvaBonus = [...]int16{0, 10, 40, 45, 68, 145, 256}
)

// piece bonuses when calulating the see
// the values are fixed to approximatively the figure bonus in mid game
var seeBonus = [FigureArraySize]int32{0, 55, 325, 341, 454, 1110, 20000}

const (
	defaultMovesToGo = 30 // default number of more moves expected to play
	infinite         = 1000000000 * time.Second
	overhead         = 20 * time.Millisecond
)

const (
	murmurMultiplier = uint64(0xc6a4a7935bd1e995)
	murmurShift      = uint(51)
)

var (
	murmurSeed = [ColorArraySize]uint64{
		0x77a166129ab66e91,
		0x4f4863d5038ea3a3,
		0xe14ec7e648a4068b,
	}
)

// end definitions
///////////////////////////////////////////////

///////////////////////////////////////////////
// init : initialization

func init() {
	// horde pawn scores
	HORDE_PAWN_SCORES[HORDE_Pawns_Side]  = Score{ M: int32(100*128) , E: int32(100*128) }
	HORDE_PAWN_SCORES[HORDE_Pieces_Side] = Score{ M: int32(100*128) , E: int32(100*128) }

	// global hash table
	GlobalHashTable = NewHashTable(DefaultHashTableSizeMB)

	// initialize caches
	pawnsAndShelterCache = newCache(9, hashPawnsAndShelter, evaluatePawnsAndShelter)
	initWeights()

	slice := func(w []Score, out []Score) []Score {
		copy(out, w)
		return w[len(out):]
	}
	entry := func(w []Score, out *Score) []Score {
		*out = w[0]
		return w[1:]
	}

	w := Weights[:]
	w = slice(w, wFigure[:])
	w = slice(w, wMobility[:])
	w = slice(w, wPawn[:])
	w = slice(w, wPassedPawn[:])
	w = slice(w, wKingRank[:])
	w = slice(w, wKingFile[:])
	w = entry(w, &wConnectedPawn)
	w = entry(w, &wDoublePawn)
	w = entry(w, &wIsolatedPawn)
	w = entry(w, &wPawnThreat)
	w = entry(w, &wKingShelter)
	w = entry(w, &wBishopPair)
	w = entry(w, &wRookOnOpenFile)
	w = entry(w, &wRookOnHalfOpenFile)

	if len(w) != 0 {
		panic(fmt.Sprintf("not all weights used, left with %d out of %d", len(w), len(Weights)))
	}
}

///////////////////////////////////////////////
// murmuxMix : mixes two integers k&h
// murmurMix is based on MurmurHash2 https://sites.google.com/site/murmurhash/
// which is on public domain
// a hash can be constructed like this:
//     hash := murmurSeed[us]
//     hash = murmurMix(hash, n1)
//     hash = murmurMix(hash, n2)
//     hash = murmurMix(hash, n3)
// -> k unint64 : k
// -> h unint64 : h
// <- uint64 : h

func murmurMix(k, h uint64) uint64 {
	h ^= k
	h *= murmurMultiplier
	h ^= h >> murmurShift
	return h
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// hashPawnsAndShelter : has pawn and shelter
// -> pos *Position : position
// -> us Color : color
// <- uint64 : h

func hashPawnsAndShelter(pos *Position, us Color) uint64 {
	h := murmurSeed[us]
	h = murmurMix(h, uint64(pos.ByPiece(us, Pawn)))
	h = murmurMix(h, uint64(pos.ByPiece(us.Opposite(), Pawn)))
	h = murmurMix(h, uint64(pos.ByPiece(us, King)))
	if pos.ByPiece(us.Opposite(), Queen) != 0 {
		// Mixes in something to signal queen's presence.
		h = murmurMix(h, murmurSeed[NoColor])
	}
	return h
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// evaluatePawns : evaluate pawns
// -> pos *Position : position
// -> us Color : color
// <- Eval : eval

func evaluatePawns(pos *Position, us Color) Eval {
	var eval Eval
	ours := pos.ByPiece(us, Pawn)
	theirs := pos.ByPiece(us.Opposite(), Pawn)

	// from white's POV (P - white pawn, p - black pawn)
	// block   wings
	// ....... .....
	// .....P. .....
	// .....x. .....
	// ..p..x. .....
	// .xxx.x. .xPx.
	// .xxx.x. .....
	// .xxx.x. .....
	// .xxx.x. .....
	block := East(theirs) | theirs | West(theirs)
	wings := East(ours) | West(ours)
	double := Bitboard(0)
	if us == White {
		block = SouthSpan(block) | SouthSpan(ours)
		double = ours & South(ours)
	} else /* if us == Black */ {
		block = NorthSpan(block) | NorthSpan(ours)
		double = ours & North(ours)
	}

	isolated := ours &^ Fill(wings)                           // no pawn on the adjacent files
	connected := ours & (North(wings) | wings | South(wings)) // has neighbouring pawns
	passed := ours &^ block                                   // no pawn env front and no enemy on the adjacent files

	for bb := ours; bb != 0; {
		sq := bb.Pop()
		povSq := sq.POV(us)
		rank := povSq.Rank()

		eval.Add(wFigure[Pawn])
		eval.Add(wPawn[povSq-8])

		if passed.Has(sq) {
			eval.Add(wPassedPawn[rank])
		}
		if connected.Has(sq) {
			eval.Add(wConnectedPawn)
		}
		if double.Has(sq) {
			eval.Add(wDoublePawn)
		}
		if isolated.Has(sq) {
			eval.Add(wIsolatedPawn)
		}
	}

	return eval
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// evaluateShelter : evaluate shelter
// -> pos *Position : position
// -> us Color : color
// <- Eval : eval

func evaluateShelter(pos *Position, us Color) Eval {
	var eval Eval
	pawns := pos.ByPiece(us, Pawn)
	king := pos.ByPiece(us, King)

	sq := king.AsSquare().POV(us)
	eval.Add(wKingFile[sq.File()])
	eval.Add(wKingRank[sq.Rank()])

	if pos.ByPiece(us.Opposite(), Queen) != 0 {
		king = ForwardSpan(us, king)
		file := sq.File()
		if file > 0 && West(king)&pawns == 0 {
			eval.Add(wKingShelter)
		}
		if king&pawns == 0 {
			eval.AddN(wKingShelter, 2)
		}
		if file < 7 && East(king)&pawns == 0 {
			eval.Add(wKingShelter)
		}
	}
	return eval
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// evaluatePawnsAndShelter : evaluate pawn and shelter
// -> pos *Position : position
// -> us Color : color
// <- Eval : eval

func evaluatePawnsAndShelter(pos *Position, us Color) Eval {
	var eval Eval
	eval.Merge(evaluatePawns(pos, us))
	eval.Merge(evaluateShelter(pos, us))
	return eval
}

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
// Put : inserts a new entry, ignores NullMoves
// -> pv *pvTable : pv table
// -> pos *Position : position
// -> move Move : move

func (pv *pvTable) Put(pos *Position, move Move) {
	if move == NullMove {
		return
	}

	// based on pos.Zobrist() two entries are looked up
	// if any of the two entries in the table matches
	// current position, then that one is replaced
	// otherwise, the older is replaced

	entry1 := &pv.table[uint32(pos.Zobrist())&pvTableMask]
	entry2 := &pv.table[uint32(pos.Zobrist()>>32)&pvTableMask]
	zobrist := pos.Zobrist()

	var entry *pvEntry
	if entry1.lock == zobrist {
		entry = entry1
	} else if entry2.lock == zobrist {
		entry = entry2
	} else if entry1.birth <= entry2.birth {
		entry = entry1
	} else {
		entry = entry2
	}

	pv.timer++
	*entry = pvEntry{
		lock:  pos.Zobrist(),
		move:  move,
		birth: pv.timer,
	}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// get : get move from pv table
// TODO: lookup move in transposition table if none is available
// -> pv *pvTable : pv table
// -> pos *Position : position
// <- Move : move

func (pv *pvTable) get(pos *Position) Move {
	entry1 := &pv.table[uint32(pos.Zobrist())&pvTableMask]
	entry2 := &pv.table[uint32(pos.Zobrist()>>32)&pvTableMask]
	zobrist := pos.Zobrist()

	var entry *pvEntry
	if entry1.lock == zobrist {
		entry = entry1
	}
	if entry2.lock == zobrist {
		entry = entry2
	}
	if entry == nil {
		return NullMove
	}

	return entry.move
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// Get : returns the principal variation
// -> pv *pvTable : pv table
// -> pos *Position : position
// <- Move[] : principal variation

func (pv *pvTable) Get(pos *Position) []Move {
	seen := make(map[uint64]bool)
	var moves []Move

	// extract the moves by following the position
	next := pv.get(pos)
	for next != NullMove && !seen[pos.Zobrist()] {
		seen[pos.Zobrist()] = true
		moves = append(moves, next)
		pos.DoMove(next)
		next = pv.get(pos)
	}

	// undo all moves, so we get back to the initial state
	for i := len(moves) - 1; i >= 0; i-- {
		pos.UndoMove()
	}
	return moves
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// passed : returns true if a passed pawn appears or disappears
// TODO: the heuristic is incomplete and doesn't handled discovered passed pawns
// -> pos *Position : position
// -> m Move : move
// <- bool : true if passed

func passed(pos *Position, m Move) bool {
	if m.Piece().Figure() == Pawn {
		// checks no pawns are in front on its and adjacent files
		bb := m.To().Bitboard()
		bb = West(bb) | bb | East(bb)
		pawns := pos.ByFigure[Pawn] &^ m.To().Bitboard() &^ m.From().Bitboard()
		if ForwardSpan(m.SideToMove(), bb)&pawns == 0 {
			return true
		}
	}
	if m.Capture().Figure() == Pawn {
		// checks no pawns are in front on its and adjacent files
		bb := m.To().Bitboard()
		bb = West(bb) | bb | East(bb)
		pawns := pos.ByFigure[Pawn] &^ m.To().Bitboard() &^ m.From().Bitboard()
		if BackwardSpan(m.SideToMove(), bb)&pawns == 0 {
			return true
		}
	}
	return false
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// isFutile : return true if m cannot raise current static
// evaluation above α, this is just an heuristic and mistakes
// can happen
// -> pos *Position : position
// -> static int32 : static
// -> α int32 : alpha
// -> margin int32 : margin
// -> m Move : move
// <- bool : true if futile

func isFutile(pos *Position, static, α, margin int32, m Move) bool {
	if m.MoveType() == Promotion {
		// promotion and passed pawns can increase static evaluation
		// by more than futilityMargin
		return false
	}
	f := m.Capture().Figure()
	δ := ScaleToCentiPawn(max(wFigure[f].M, wFigure[f].E))
	return static+δ+margin < α && !passed(pos, m)
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
	if IS_Racing_Kings {
		for i:=Knight; i<King ; i++ {
			fmt.Printf("%s %d\n",FigureToName[i],RK_PIECE_VALUES[i])
		}
		fmt.Printf("King Advance %d\n",KING_ADVANCE_VALUE)
	}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// SetVariantFlags : set variant flags

func SetVariantFlags() {
	IS_Standard = false
	IS_Racing_Kings = false
	IS_Atomic = false
	IS_Horde = false
	if Variant == VARIANT_Standard {
		IS_Standard = true
	}
	if Variant == VARIANT_Racing_Kings {
		IS_Racing_Kings = true
	}
	if Variant == VARIANT_Atomic {
		IS_Atomic = true
	}
	if Variant == VARIANT_Horde {
		IS_Horde = true
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
	SetVariantFlags()
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
	///////////////////////////////////////////////
	// NEW
	// in atomic increase mobility score
	if IS_Atomic {
		n = n * 10
	}
	// END NEW
	///////////////////////////////////////////////
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
// Multiply : multiply score by constant
// -> score Score : score
// -> n int : multiplication factor
// <- Score : multiplied score

func (score Score) Multiply(n int32) Score {
	return Score{ M: n * score.M, E: n * score.E }
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// evaluateSide : evaluates position for a single side
// -> pos *Position : position
// -> us Color : us
// -> eval *Eval : eval

func evaluateSide(pos *Position, us Color, eval *Eval) {
	if !IS_Horde {
		// in horde ignore this and use simply the pawn material
		eval.Merge(pawnsAndShelterCache.load(pos, us))
	} else {
		// calculate pawn material for horde
		for bb := pos.ByPiece(us, Pawn); bb > 0; {
			sq := bb.Pop()
			eval.Add(HORDE_PAWN_SCORES[us])
			eval.Add(HORDE_CENTER_BONUS.Multiply(HORDE_CENTER_BONUS_WEIGHTS[sq.File()]))
		}
		if us == HORDE_Pawns_Side {
			// add balance for pawns
			eval.Add(HORDE_BALANCE_SCORE)
		}
	}
	all := pos.ByColor[White] | pos.ByColor[Black]
	them := us.Opposite()

	// Pawn
	mobility := Forward(us, pos.ByPiece(us, Pawn)) &^ all
	eval.AddN(wMobility[Pawn], mobility.Count())
	mobility = pos.PawnThreats(us) & pos.ByColor[us.Opposite()]
	eval.AddN(wPawnThreat, mobility.Count())

	if IS_Atomic {
		// in atomic add bonus for pawns
		for bb := pos.ByPiece(us, Pawn); bb > 0; {
			bb.Pop()
			eval.Add(ATOMIC_PAWN_BONUS_SCORE)
		}
	}

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

		// evaluate rook on open and semi open files
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
	if IS_Racing_Kings {
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
	if IS_Racing_Kings {
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
	// in horde all pawns captured for the pawns side is mate
	if IS_Horde {
		if pos.AllPawnsCaptured() {
			if HORDE_Pawns_Side == White {
				return scoreMultiplier[pos.SideToMove] * (MatedScore + eng.ply()), true
			} else {
				return scoreMultiplier[pos.SideToMove] * (MateScore - eng.ply()), true
			}
		}
	}
	// trivial cases when kings are missing
	if pos.ByPiece(White, King) == 0 && pos.ByPiece(Black, King) == 0 {
		return 0, true
	}
	if pos.ByPiece(White, King) == 0 {
		mateok := true
		if IS_Horde {
			// in horde pawns having no king is not mate
			if HORDE_Pawns_Side == White {
				mateok = false
			}
		}
		if mateok {
			return scoreMultiplier[pos.SideToMove] * (MatedScore + eng.ply()), true
		}
	}
	if pos.ByPiece(Black, King) == 0 {
		mateok := true
		if IS_Horde {
			// in horde pawns having no king is not mate
			if HORDE_Pawns_Side == Black {
				mateok = false
			}
		}
		if mateok {
			return scoreMultiplier[pos.SideToMove] * (MateScore - eng.ply()), true
		}
	}
	// Neither side cannot mate.
	if pos.InsufficientMaterial() {
		if IS_Horde {
			// handle insufficient material in horde
		} else {
			return 0, true
		}
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
// NewHashTable : builds transposition table that takes up to hashSizeMB megabytes
// -> hashSizeMB int : hash size
// <- *HashTable : hash table

func NewHashTable(hashSizeMB int) *HashTable {
	// Choose hashSize such that it is a power of two.
	hashEntrySize := uint64(unsafe.Sizeof(hashEntry{}))
	hashSize := uint64(hashSizeMB) << 20 / hashEntrySize

	for hashSize&(hashSize-1) != 0 {
		hashSize &= hashSize - 1
	}
	return &HashTable{
		table: make([]hashEntry, hashSize),
		mask:  uint32(hashSize - 1),
	}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// size returns the number of entries in the table
// -> ht *HashTable : hash table
// <- int : size

func (ht *HashTable) Size() int {
	return int(ht.mask + 1)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// split : splits lock into a lock and two hash table indexes
// expects mask to be at least 3 bits
// -> lock uint64 : lock
// -> mask uint32 : mask
// <- uint32 : lock
// <- uint32 : hash table index 1
// <- uint32 : hash table index 2

func split(lock uint64, mask uint32) (uint32, uint32, uint32) {
	hi := uint32(lock >> 32)
	lo := uint32(lock)
	h0 := lo & mask
	h1 := h0 ^ (lo >> 29)
	return hi, h0, h1
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// put puts a new entry in the database
// -> ht *HashTable : hash table
// -> pos *Position : position
// -> entry hashEntry : hash entry

func (ht *HashTable) put(pos *Position, entry hashEntry) {
	lock, key0, key1 := split(pos.Zobrist(), ht.mask)
	entry.lock = lock

	if e := &ht.table[key0]; e.lock == lock || e.kind == noEntry || e.depth+1 >= entry.depth {
		ht.table[key0] = entry

	} else {
		ht.table[key1] = entry
	}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// get : returns the hash entry for position
// observation: due to collision errors, the hashEntry returned might be
// from a different table, however, these errors are not common because
// we use 32-bit lock + log_2(len(ht.table)) bits to avoid collisions
// -> ht *HashTable : hash table
// -> pos *Position : position
// <- entry hashEntry : hash entry

func (ht *HashTable) get(pos *Position) hashEntry {
	lock, key0, key1 := split(pos.Zobrist(), ht.mask)
	if ht.table[key0].lock == lock {
		return ht.table[key0]
	}
	if ht.table[key1].lock == lock {
		return ht.table[key1]
	}
	return hashEntry{}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// Clear : removes all entries from hash
// -> ht *HashTable : hash table

func (ht *HashTable) Clear() {
	for i := range ht.table {
		ht.table[i] = hashEntry{}
	}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// retrieveHash : gets from GlobalHashTable the current position
// -> eng *Engine : engine
// <- hashEntry : hash entry

func (eng *Engine) retrieveHash() hashEntry {
	entry := GlobalHashTable.get(eng.Position)

	if entry.kind == noEntry {
		eng.Stats.CacheMiss++
		return hashEntry{}
	}
	if entry.move != NullMove && !eng.Position.IsPseudoLegal(entry.move) {
		eng.Stats.CacheMiss++
		return hashEntry{}
	}

	// return mate score relative to root
	// the score was adjusted relative to position before the
	// hash table was updated
	if entry.score < KnownLossScore {
		if entry.kind == exact {
			entry.score += eng.ply()
		}
	} else if entry.score > KnownWinScore {
		if entry.kind == exact {
			entry.score -= eng.ply()
		}
	}

	eng.Stats.CacheHit++
	return entry
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// updateHash : updates GlobalHashTable with the current position
// -> eng *Engine : engine
// -> α int32 : alpha
// -> β int32 : beta
// -> depth int32 : depth
// -> score int32 : score
// -> move Move : move

func (eng *Engine) updateHash(α, β, depth, score int32, move Move) {
	kind := exact
	if score <= α {
		kind = failedLow
	} else if score >= β {
		kind = failedHigh
	}

	// save the mate score relative to the current position
	// when retrieving from hash the score will be adjusted relative to root
	if score < KnownLossScore {
		if kind == exact {
			score -= eng.ply()
		} else if kind == failedLow {
			score = KnownLossScore
		} else {
			return
		}
	} else if score > KnownWinScore {
		if kind == exact {
			score += eng.ply()
		} else if kind == failedHigh {
			score = KnownWinScore
		} else {
			return
		}
	}

	GlobalHashTable.put(eng.Position, hashEntry{
		kind:  kind,
		score: score,
		depth: int8(depth),
		move:  move,
	})
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// get : returns the moveStack for current ply
// allocates memory if necessary
// -> st *stack : stack
// <- *moveStack : move stack

func (st *stack) get() *moveStack {
	for len(st.moves) <= st.position.Ply {
		st.moves = append(st.moves, moveStack{
			moves: make([]Move, 0, 4),
			order: make([]int16, 0, 4),
		})
	}
	return &st.moves[st.position.Ply]
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// GenerateMoves : generates all moves of kind
// -> st *stack : stack
// -> kint int : kind
// -> hash Move : move

func (st *stack) GenerateMoves(kind int, hash Move) {
	ms := st.get()
	ms.moves = ms.moves[:0] // clear the array, but keep the backing memory
	ms.order = ms.order[:0]
	ms.kind = kind
	ms.state = msHash
	ms.hash = hash
	// ms.killer = ms.killer // keep killers
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// generateMoves : generates all moves
// -> st *stack : stack
// -> kint int : kind

func (st *stack) generateMoves(kind int) {
	ms := &st.moves[st.position.Ply]
	if len(ms.moves) != 0 || len(ms.order) != 0 {
		panic("expected no moves")
	}
	if ms.kind&kind == 0 {
		return
	}
	st.position.GenerateMoves(ms.kind&kind, &ms.moves)
	for _, m := range ms.moves {
		ms.order = append(ms.order, mvvlva(m))
	}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// moveBest : moves best move to front
// -> st *stack : stack

func (st *stack) moveBest() {
	ms := &st.moves[st.position.Ply]
	if len(ms.moves) == 0 {
		return
	}

	bi := 0
	for i := range ms.moves {
		if ms.order[i] > ms.order[bi] {
			bi = i
		}
	}

	last := len(ms.moves) - 1
	ms.moves[bi], ms.moves[last] = ms.moves[last], ms.moves[bi]
	ms.order[bi], ms.order[last] = ms.order[last], ms.order[bi]
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// popFront : pops the move from the front
// -> st *stack : stack
// <- Move : move

func (st *stack) popFront() Move {
	ms := &st.moves[st.position.Ply]
	if len(ms.moves) == 0 {
		return NullMove
	}

	last := len(ms.moves) - 1
	move := ms.moves[last]
	ms.moves = ms.moves[:last]
	ms.order = ms.order[:last]
	return move
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// Pop : pops a new move
// returns NullMove if there are no moves
// moves are generated in several phases:
//	first the hash move,
//  then the violent moves,
//  then the killer moves,
//  then the tactical and quiet moves
// -> st *stack : stack
// <- Move : move

func (st *stack) PopMove() Move {
	ms := &st.moves[st.position.Ply]
	for {
		switch ms.state {
		// return the hash move
		case msHash:
			// return the hash move directly without generating the pseudo legal moves
			ms.state = msGenViolent
			if st.position.IsPseudoLegal(ms.hash) {
				return ms.hash
			}

		// return the violent moves
		case msGenViolent:
			ms.state = msReturnViolent
			st.generateMoves(Violent)

		case msReturnViolent:
			// most positions have only very violent moves so
			// it doesn't make sense to sort given that captures have a high
			// chance to fail high, we just pop the moves in order of score
			st.moveBest()
			if m := st.popFront(); m == NullMove {
				if ms.kind&(Tactical|Quiet) == 0 {
					// optimization: skip remaining steps if no Tactical or Quiet moves
					// were requested (e.g. in quiescence search)
					ms.state = msDone
				} else {
					ms.state = msGenKiller
				}
			} else if m == ms.hash {
				break
			} else if m != NullMove {
				return m
			}

		// return killer moves
		// NB: not all killer moves are valid
		case msGenKiller:
			ms.state = msReturnKiller
			for i := len(ms.killer) - 1; i >= 0; i-- {
				if m := ms.killer[i]; m != NullMove {
					ms.moves = append(ms.moves, ms.killer[i])
					ms.order = append(ms.order, -int16(i))
				}
			}

		case msReturnKiller:
			if m := st.popFront(); m == NullMove {
				ms.state = msGenRest
			} else if m == ms.hash {
				break
			} else if st.position.IsPseudoLegal(m) {
				return m
			}

		// return the quiet and tactical moves in the order they were generated
		case msGenRest:
			ms.state = msReturnRest
			st.generateMoves(Tactical | Quiet)

		case msReturnRest:
			if m := st.popFront(); m == NullMove {
				ms.state = msDone
			} else if m == ms.hash || st.IsKiller(m) {
				break
			} else {
				return m
			}

		case msDone:
			// Just in case another move is requested
			return NullMove
		}

	}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// IsKiller : returns true if m is a killer move for currenty ply
// -> st *stack : stack
// -> m Move : move
// <- bool : true if killer move

func (st *stack) IsKiller(m Move) bool {
	ms := &st.moves[st.position.Ply]
	return m == ms.killer[0] || m == ms.killer[1] || m == ms.killer[2] || m == ms.killer[3]
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// SaveKiller : saves a killer move, m
// -> st *stack : stack
// -> m Move : move

func (st *stack) SaveKiller(m Move) {
	ms := &st.moves[st.position.Ply]
	if !m.IsViolent() {
		// Move the newly found killer first.
		if m == ms.killer[0] {
			// do nothing
		} else if m == ms.killer[1] {
			ms.killer[1] = ms.killer[0]
			ms.killer[0] = m
		} else if m == ms.killer[2] {
			ms.killer[2] = ms.killer[1]
			ms.killer[1] = ms.killer[0]
			ms.killer[0] = m
		} else {
			ms.killer[3] = ms.killer[2]
			ms.killer[2] = ms.killer[1]
			ms.killer[1] = ms.killer[0]
			ms.killer[0] = m
		}
	}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// mvvlva : computes Most Valuable Victim / Least Valuable Aggressor
// https://chessprogramming.wikispaces.com/MVV-LVA
// -> m Move : move
// <- int16 : MVV-LVA

func mvvlva(m Move) int16 {
	a := int(m.Target().Figure())
	v := int(m.Capture().Figure())
	return int16(mvvlvaBonus[v]*64 - mvvlvaBonus[a])
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// Reset : clear the stack for a new position
// -> st *stack : stack
// -> pos *Position : position

func (st *stack) Reset(pos *Position) {
	st.position = pos
	st.moves = st.moves[:0]
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// seeScore : see score
// -> m Move : move
// <- int32 : score

func seeScore(m Move) int32 {
	score := seeBonus[m.Capture().Figure()]
	if m.MoveType() == Promotion {
		score -= seeBonus[Pawn]
		score += seeBonus[m.Target().Figure()]
	}
	return score
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// seeSign : return true if see(m) < 0
// -> pos *Position : position
// -> m Move : move
// <- bool : true if see(m) < 0

func seeSign(pos *Position, m Move) bool {
	if m.Piece().Figure() <= m.Capture().Figure() {
		// Even if m.Piece() is captured, we are still positive.
		return false
	}
	return see(pos, m) < 0
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// see : returns the static exchange evaluation for m, where is
// the last move executed
// https://chessprogramming.wikispaces.com/Static+Exchange+Evaluation
// https://chessprogramming.wikispaces.com/SEE+-+The+Swap+Algorithm
// the implementation here is optimized for the common case when there
// isn't any capture following the move, the score returned is based
// on some fixed values for figures, different from the ones
// defined in material.go
// -> pos *Position : position
// -> m Move : move
// <- int32 : score

func see(pos *Position, m Move) int32 {
	us := pos.SideToMove
	sq := m.To()
	bb := sq.Bitboard()
	target := m.Target() // piece in position
	bb27 := bb &^ (BbRank1 | BbRank8)
	bb18 := bb & (BbRank1 | BbRank8)

	// cccupancy tables as if moves are executed
	var occ [ColorArraySize]Bitboard
	occ[White] = pos.ByColor[White]
	occ[Black] = pos.ByColor[Black]
	all := occ[White] | occ[Black]

	// adjust score for move
	score := seeScore(m)
	tmp := [16]int32{score}
	gain := tmp[:1]

	for score >= 0 {
		// try every figure in order of value
		var fig Figure                  // attacking figure
		var att Bitboard                // attackers
		var pawn, bishop, rook Bitboard // mobilies for our figures

		ours := occ[us]
		mt := Normal

		// pawn attacks
		pawn = Backward(us, West(bb27)|East(bb27))
		if att = pawn & ours & pos.ByFigure[Pawn]; att != 0 {
			fig = Pawn
			goto makeMove
		}

		if att = bbKnightAttack[sq] & ours & pos.ByFigure[Knight]; att != 0 {
			fig = Knight
			goto makeMove
		}

		if bbSuperAttack[sq]&ours == 0 {
			// no other figure can attack sq so we give up early
			break
		}

		bishop = BishopMobility(sq, all)
		if att = bishop & ours & pos.ByFigure[Bishop]; att != 0 {
			fig = Bishop
			goto makeMove
		}

		rook = RookMobility(sq, all)
		if att = rook & ours & pos.ByFigure[Rook]; att != 0 {
			fig = Rook
			goto makeMove
		}

		// pawn promotions are considered queens minus the pawn
		pawn = Backward(us, West(bb18)|East(bb18))
		if att = pawn & ours & pos.ByFigure[Pawn]; att != 0 {
			fig, mt = Queen, Promotion
			goto makeMove
		}

		if att = (rook | bishop) & ours & pos.ByFigure[Queen]; att != 0 {
			fig = Queen
			goto makeMove
		}

		if att = bbKingAttack[sq] & ours & pos.ByFigure[King]; att != 0 {
			fig = King
			goto makeMove
		}

		// no attack found
		break

	makeMove:
		// make a new pseudo-legal move of the smallest attacker
		from := att.LSB()
		attacker := ColorFigure(us, fig)
		m := MakeMove(mt, from.AsSquare(), sq, target, attacker)
		target = attacker // attacker becomes the new target

		// update score
		score = seeScore(m) - score
		gain = append(gain, score)

		// update occupancy tables for executing the move
		occ[us] = occ[us] &^ from
		all = all &^ from

		// switch sides
		us = us.Opposite()
	}

	for i := len(gain) - 2; i >= 0; i-- {
		if -gain[i+1] < gain[i] {
			gain[i] = -gain[i+1]
		}
	}
	return gain[0]
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// searchQuiescence : evaluates the position after solving all captures
// this is a very limited search which considers only violent moves
// checks are not considered, in fact it assumes that the move
// ordering will always put the king capture first
// -> eng *Engine : engine
// -> α int32 : alpha
// -> β int32 : beta
// <- int32 : score

func (eng *Engine) searchQuiescence(α, β int32) int32 {

	if IS_Atomic {
		// quiescence problematic in atomic because of captures cause explosions
		return eng.Score()
	}

	eng.Stats.Nodes++
	if score, done := eng.endPosition(); done {
		return score
	}

	// stand pat
	// TODO: some suggest to not stand pat when in check
	// however, I did several tests and handling checks in quiescence
	// doesn't help at all
	static := eng.Score()
	if static >= β {
		return static
	}
	localα := α
	if static > localα {
		localα = static
	}

	pos := eng.Position
	us := pos.SideToMove
	inCheck := pos.IsChecked(us)

	var bestMove Move
	eng.stack.GenerateMoves(Violent, NullMove)
	for move := eng.stack.PopMove(); move != NullMove; move = eng.stack.PopMove() {
		// prune futile moves that would anyway result in a stand-pat
		// at that next depth
		if !inCheck && isFutile(pos, static, localα, futilityMargin, move) {
			// TODO: should it update localα?
			continue
		}

		// discard illegal or losing captures
		eng.DoMove(move)
		if eng.Position.IsChecked(us) ||
			!inCheck && move.MoveType() == Normal && seeSign(pos, move) {
			eng.UndoMove()
			continue
		}

		///////////////////////////////////////////////////
		// NEW
		// in Racing Kings avoid captures that give check
		if IS_Racing_Kings {
			if eng.Position.IsCheckedLocal(us.Opposite()) {
				eng.UndoMove()
				continue
			}
		}
		// END NEW
		///////////////////////////////////////////////////

		score := -eng.searchQuiescence(-β, -localα)
		eng.UndoMove()

		if score >= β {
			return score
		}
		if score > localα {
			localα = score
			bestMove = move
		}
	}

	if α < localα && localα < β {
		eng.pvTable.Put(eng.Position, bestMove)
	}
	return localα
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// set : set atomic flag
// -> af *atomicFlag

func (af *atomicFlag) set() {
	af.lock.Lock()
	af.flag = true
	af.lock.Unlock()
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// set : get atomic flag
// -> af *atomicFlag
// <- bool : value of atomic flag

func (af *atomicFlag) get() bool {
	af.lock.Lock()
	tmp := af.flag
	af.lock.Unlock()
	return tmp
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// NewTimeControl : returns a new time control with no time limit,
// no depth limit, zero time increment and zero moves to go
// -> pos *Position : position
// -> predicted bool : predicted
// <- *TimeControl : time control

func NewTimeControl(pos *Position, predicted bool) *TimeControl {
	// branch more when there are more pieces. With fewer pieces
	// there is less mobility and hash table kicks in more often
	branch := 2
	for np := (pos.ByColor[White] | pos.ByColor[Black]).Count(); np > 0; np /= 6 {
		branch++
	}

	return &TimeControl{
		WTime:      infinite,
		WInc:       0,
		BTime:      infinite,
		BInc:       0,
		Depth:      64,
		MovesToGo:  defaultMovesToGo,
		sideToMove: pos.SideToMove,
		predicted:  predicted,
		branch:     branch,
	}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// NewFixedDepthTimeControl : returns a TimeControl which limits the search depth
// -> pos *Position : position
// -> depth int32 : depth
// <- *TimeControl : time control

func NewFixedDepthTimeControl(pos *Position, depth int32) *TimeControl {
	tc := NewTimeControl(pos, false)
	tc.Depth = depth
	tc.MovesToGo = 1
	return tc
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// NewDeadlineTimeControl : returns a TimeControl corresponding to a single move before deadline
// -> pos *Position : position
// -> deadline time.Duration : deadline
// <- *TimeControl : time control

func NewDeadlineTimeControl(pos *Position, deadline time.Duration) *TimeControl {
	tc := NewTimeControl(pos, false)
	tc.WTime = deadline
	tc.BTime = deadline
	tc.MovesToGo = 1
	return tc
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// thinkingTime : calculates how much time to think this round
// t is the remaining time, i is the increment
// -> tc *TimeControl : time control
// <- time.Duration : think time

func (tc *TimeControl) thinkingTime() time.Duration {
	// the formula allows engine to use more of time in the begining
	// and rely more on the increment later
	tmp := time.Duration(tc.MovesToGo)
	tt := (tc.time + (tmp-1)*tc.inc) / tmp

	if tt < 0 {
		return 0
	}
	if tc.predicted {
		tt = tt * 4 / 3
	}
	if tt < tc.limit {
		return tt
	}
	return tc.limit
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// Start : starts the timer
// should start as soon as possible to set the correct time
// -> tc *TimeControl : time control
// -> ponder bool : ponder

func (tc *TimeControl) Start(ponder bool) {
	if tc.sideToMove == White {
		tc.time, tc.inc = tc.WTime, tc.WInc
	} else {
		tc.time, tc.inc = tc.BTime, tc.BInc
	}

	// calcuates the last moment when the search should be stopped
	if tc.time > 2*overhead {
		tc.limit = tc.time - overhead
	} else if tc.time > overhead {
		tc.limit = overhead
	} else {
		tc.limit = tc.time
	}

	// increase the branchFactor a bit to be on the
	// safe side when there are only a few moves left
	for i := 4; i > 0; i /= 2 {
		if tc.MovesToGo <= i {
			tc.branch++
		}
	}

	tc.stopped = atomicFlag{flag: false}
	tc.ponderhit = atomicFlag{flag: !ponder}

	tc.searchTime = tc.thinkingTime()
	tc.updateDeadlines() // deadlines are ignored while pondering (ponderHit == false)
}

///////////////////////////////////////////////
// updateDeadlines : update deadline
// -> tc *TimeControl : time control

func (tc *TimeControl) updateDeadlines() {
	now := time.Now()
	tc.searchDeadline = now.Add(tc.searchTime / time.Duration(tc.branch))

	// stopDeadline is when to abort the search in case of an explosion
	// we give a large overhead here so the search is not aborted very often
	deadline := tc.searchTime * 4
	if deadline > tc.limit {
		deadline = tc.limit
	}
	tc.stopDeadline = now.Add(deadline)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// NextDepth : returns true if search can start at depth
// in any case Stopped() will return false
// -> tc *TimeControl : time control
// -> depth int32 : depth
// <- bool : true if search can start at depth

func (tc *TimeControl) NextDepth(depth int32) bool {
	tc.currDepth = depth
	return tc.currDepth <= tc.Depth && !tc.hasStopped(tc.searchDeadline)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// PonderHit : switch to our time control
// -> tc *TimeControl : time control

func (tc *TimeControl) PonderHit() {
	tc.updateDeadlines()
	tc.ponderhit.set()
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// Stop : marks the search as stopped
// -> tc *TimeControl : time control

func (tc *TimeControl) Stop() {
	tc.stopped.set()
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// hasStopped : has stopped
// -> tc *TimeControl : time control
// -> deadline time.Time : deadline
// <- bool : true if stopped

func (tc *TimeControl) hasStopped(deadline time.Time) bool {
	if tc.currDepth <= 2 {
		// run for at few depths at least otherwise mates can be missed
		return false
	}
	if tc.stopped.get() {
		// use a cached value if available
		return true
	}
	if tc.ponderhit.get() && time.Now().After(deadline) {
		// stop search if no longer pondering and deadline as passed
		return true
	}
	return false
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// Stopped : returns true if the search has stopped because
// Stop() was called or the time has ran out
// -> tc *TimeControl : time control
// <- bool : true if stopped

func (tc *TimeControl) Stopped() bool {
	if !tc.hasStopped(tc.stopDeadline) {
		return false
	}
	// time has ran out so flip the stopped flag
	tc.stopped.set()
	return true
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// searchTree : implements searchTree framework
// searchTree fails soft, i.e. the score returned can be outside the bounds
// invariants:
//   if score <= α then the search failed low and the score is an upper bound
//   else if score >= β then the search failed high and the score is a lower bound
//   else score is exact
// assuming this is a maximizing nodes, failing high means that a
// minimizing ancestor node already has a better alternative
// -> eng *Engine : engine
// -> α int32 : alpha, lower bound
// -> β int32 : beta, upper bound
// -> depth int32 : remaining depth (decreasing)
// -> ignoremoves []Move : list of moves that should not be searched
// <- int32 : score of the current position up to depth (modulo reductions/extensions) from current player's POV

func (eng *Engine) searchTree(α, β, depth int32, ignoremoves []Move) int32 {
	ply := eng.ply()
	pvNode := α+1 < β
	pos := eng.Position
	us := pos.SideToMove

	// update statistics
	eng.Stats.Nodes++
	if !eng.stopped && eng.Stats.Nodes >= eng.checkpoint {
		eng.checkpoint = eng.Stats.Nodes + checkpointStep
		if eng.timeControl.Stopped() {
			eng.stopped = true
		}
	}
	if eng.stopped {
		return α
	}
	if pvNode && ply > eng.Stats.SelDepth {
		eng.Stats.SelDepth = eng.ply()
	}

	// verify that this is not already an endgame
	if score, done := eng.endPosition(); done {
		return score
	}

	// mate pruning: if an ancestor already has a mate in ply moves then
	// the search will always fail low so we return the lowest wining score
	if MateScore-ply <= α {
		return KnownWinScore
	}

	// check the transposition table
	entry := eng.retrieveHash()
	hash := entry.move
	if entry.kind != noEntry && depth <= int32(entry.depth) {
		if entry.kind == exact {
			// simply return if the score is exact
			// update principal variation table if possible
			if α < entry.score && entry.score < β {
				eng.pvTable.Put(pos, hash)
			}
			return entry.score
		}
		if entry.kind == failedLow && entry.score <= α {
			// previously the move failed low so the actual score
			// is at most entry.score, uf that's lower than α
			// this will also fail low
			return entry.score
		}
		if entry.kind == failedHigh && entry.score >= β {
			// previously the move failed high so the actual score
			// is at least entry.score, if that's higher than β
			// this will also fail high
			return entry.score
		}
	}

	// stop searching when the maximum search depth is reached
	if depth <= 0 {
		// depth can be < 0 due to aggressive LMR
		score := eng.searchQuiescence(α, β)
		eng.updateHash(α, β, depth, score, NullMove)
		return score
	}

	sideIsChecked := pos.IsChecked(us)

	// do a null move, if the null move fails high then the current
	// position is too good, so opponent will not play it
	// verification that we are not in check is done by tryMove
	// which bails out if after the null move we are still in check
	if depth > NullMoveDepthLimit && // not very close to leafs
		!sideIsChecked && // nullmove is illegal when in check
		pos.HasNonPawns(us) && // at least one minor/major piece
		KnownLossScore < α && β < KnownWinScore { // disable in lost or won positions

		reduction := NullMoveDepthReduction
		if pos.NumNonPawns(us) >= 3 {
			// reduce more when there are three minor/major pieces
			reduction++
		}

		eng.DoMove(NullMove)
		score := eng.tryMove(β-1, β, depth-reduction, 0, false, NullMove)
		if score >= β {
			return score
		}
	}

	bestMove, bestScore := NullMove, -InfinityScore

	// futility and history pruning at frontier nodes
	// based on Deep Futility Pruning http://home.hccnet.nl/h.g.muller/deepfut.html
	// based on History Leaf Pruning https://chessprogramming.wikispaces.com/History+Leaf+Pruning
	static := int32(0)
	allowLeafsPruning := false
	if depth <= FutilityDepthLimit && // enable when close to the frontier
		!sideIsChecked && // disable in check
		!pvNode && // disable in pv nodes
		KnownLossScore < α && β < KnownWinScore { // disable when searching for a mate
		allowLeafsPruning = true
		static = eng.Score()
	}

	// principal variation search: search with a null window if there is already a good move
	nullWindow := false // updated once alpha is improved
	// late move reduction: search best moves with full depth, reduce remaining moves
	allowLateMove := !sideIsChecked && depth > LMRDepthLimit

	// dropped true if not all moves were searched
	// mate cannot be declared unless all moves were tested
	dropped := false
	numQuiet := int32(0)
	localα := α

	eng.stack.GenerateMoves(All, hash)
	for move := eng.stack.PopMove(); move != NullMove; move = eng.stack.PopMove() {
		// skip moves that are on the ignore list
		if len(ignoremoves) > 0 {
			found := false
			for _ , im := range ignoremoves {
				if move == im {
					found = true
					break
				}
			}
			if found {
				continue
			}
		}

		critical := move == hash || eng.stack.IsKiller(move)
		if move.IsQuiet() {
			numQuiet++ // TODO: move from here
		}

		newDepth := depth
		eng.DoMove(move)

		// skip illegal moves that leave the king in check
		if pos.IsChecked(us) {
			eng.UndoMove()
			continue
		}

		///////////////////////////////////////////////////
		// NEW
		// in Racing Kings skip moves that give check
		if IS_Racing_Kings {
			if pos.IsCheckedLocal(us.Opposite()) {
				eng.UndoMove()
				continue
			}
		}
		///////////////////////////////////////////////////

		// extend the search when our move gives check
		// however do not extend if we can just take the undefended piece
		// see discussion: http://www.talkchess.com/forum/viewtopic.php?t=56361
		// when the move gives check, history pruning and futility pruning are also disabled
		givesCheck := pos.IsChecked(us.Opposite())
		if givesCheck {
			if pos.GetAttacker(move.To(), us.Opposite()) == NoFigure ||
				pos.GetAttacker(move.To(), us) != NoFigure {
				newDepth += CheckDepthExtension
			}
		}

		// reduce late quiet moves and bad captures
		// TODO: do not compute see when in check
		lmr := int32(0)
		if allowLateMove && !givesCheck && !critical {
			if move.IsQuiet() {
				// reduce quiet moves more at high depths and after many quiet moves
				// large numQuiet means it's likely not a CUT node
				// large depth means reductions are less risky
				lmr = 1 + min(depth, numQuiet)/5
			} else if seeSign(pos, move) {
				// bad captures (SEE<0) can be reduced, too
				lmr = 1
			}
		}

		// prune moves close to frontier
		if allowLeafsPruning && !givesCheck && !critical {
			// prune quiet moves that performed bad historically
			if bad, good := eng.history.get(move); bad > 16*good && (move.IsQuiet() || seeSign(pos, move)) {
				dropped = true
				eng.UndoMove()
				continue
			}
			// prune moves that do not raise alphas
			if isFutile(pos, static, localα, depth*futilityMargin, move) {
				bestScore = max(bestScore, static)
				dropped = true
				eng.UndoMove()
				continue
			}
		}

		score := eng.tryMove(localα, β, newDepth, lmr, nullWindow, move)
		if allowLeafsPruning && !givesCheck { // update history scores
			eng.history.inc(move, score > α)
		}
		if score >= β { // fail high, cut node
			eng.stack.SaveKiller(move)
			eng.updateHash(α, β, depth, score, move)
			return score
		}
		if score > bestScore {
			nullWindow = true
			bestMove, bestScore = move, score
			localα = max(localα, score)
		}
	}

	if !dropped {
		// if no move was found then the game is over
		if bestMove == NullMove {
			if sideIsChecked {
				bestScore = MatedScore + ply
			} else {
				bestScore = 0
			}
		}
		// update hash and principal variation tables
		eng.updateHash(α, β, depth, bestScore, bestMove)
		if α < bestScore && bestScore < β {
			eng.pvTable.Put(pos, bestMove)
		}
	}

	return bestScore
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// tryMove : makes a move and descends on the search tree
// -> eng *Engine : engine
// -> α int32 : alpha, lower bound
// -> β int32 : beta, upper bound
// -> depth int32 : remaining depth (decreasing)
// -> lmr int32 : how much to reduce a late move, Implies non-null move
// -> nullWindow bool : indicates whether to scout first, implies non-null move
// -> move Move : move to execute, can be NullMove
// <- int32 : score from the deeper search

func (eng *Engine) tryMove(α, β, depth, lmr int32, nullWindow bool, move Move) int32 {
	depth--

	score := α + 1
	if lmr > 0 { // reduce late moves
		score = -eng.searchTree(-α-1, -α, depth-lmr, []Move{})
	}

	if score > α { // if late move reduction is disabled or has failed
		if nullWindow {
			score = -eng.searchTree(-α-1, -α, depth, []Move{})
			if α < score && score < β {
				score = -eng.searchTree(-β, -α, depth, []Move{})
			}
		} else {
			score = -eng.searchTree(-β, -α, depth, []Move{})
		}
	}

	eng.UndoMove()
	return score
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// search : starts the search up to depth depth
// -> eng *Engine : engine
// -> depth int32 : depth
// -> estimated int32 : the score from previous depths
// -> ignoremoves []Move : list of moves that should not be searched
// <- int32 : score from current side to move POV

func (eng *Engine) search(depth, estimated int32, ignoremoves []Move) int32 {
	// this method only implements aspiration windows
	// the gradual widening algorithm is the one used by RobboLito
	// and Stockfish and it is explained here:
	// http://www.talkchess.com/forum/viewtopic.php?topic_view=threads&p=499768&t=46624
	γ, δ := estimated, int32(initialAspirationWindow)
	α, β := max(γ-δ, -InfinityScore), min(γ+δ, InfinityScore)
	score := estimated

	if depth < 4 {
		// disable aspiration window for very low search depths
		// this wastes a lot of time when for tunning
		α = -InfinityScore
		β = +InfinityScore
	}

	for !eng.stopped {
		// at root a non-null move is required, cannot prune based on null-move
		score = eng.searchTree(α, β, depth, ignoremoves)
		if score <= α {
			α = max(α-δ, -InfinityScore)
			δ += δ / 2
		} else if score >= β {
			β = min(β+δ, InfinityScore)
			δ += δ / 2
		} else {
			return score
		}
	}

	return score
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// Play : evaluates current position
// returns the 
// -> eng *Engine : engine
// -> tc *TimeControl : time control, should already be started
// -> ignoremoves []Move : list of moves that shold be ignored in search
// <- moves []Move : principal variation, that is
//  moves[0] is the best move found and
//  moves[1] is the pondering move
//  if no move was found because the game has finished
//  then an empty pv is returned

func (eng *Engine) Play(tc *TimeControl, ignoremoves []Move) (moves []Move) {
	eng.Log.BeginSearch()
	eng.Stats = Stats{Depth: -1}

	eng.rootPly = eng.Position.Ply
	eng.timeControl = tc
	eng.stopped = false
	eng.checkpoint = checkpointStep
	eng.stack.Reset(eng.Position)

	score := int32(0)
	for depth := int32(0); depth < 64; depth++ {
		if !tc.NextDepth(depth) {
			// stop if tc control says we are done
			// search at least one depth, otherwise a move cannot be returned
			break
		}

		eng.Stats.Depth = depth
		score = eng.search(depth, score, ignoremoves)

		if !eng.stopped {
			// if eng has not been stopped then this is a legit pv
			moves = eng.pvTable.Get(eng.Position)
			eng.Log.PrintPV(eng.Stats, score, moves)
		}
	}

	eng.Log.EndSearch()
	return moves
}

///////////////////////////////////////////////