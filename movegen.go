//////////////////////////////////////////////////////
// movegen.go
// everything that is necessary for move generation
// zurichess sources: basic.go, misc.go
//////////////////////////////////////////////////////

package lib

// imports

import(
	"fmt"
)

// Figure represents a piece without a color
type Figure uint

const (
	NoFigure Figure = iota
	Pawn
	Knight
	Bishop
	Rook
	Queen
	King

	FigureArraySize = int(iota)
	FigureMinValue  = Pawn
	FigureMaxValue  = King
)

// figure to symbol
var (
	figureToSymbol = map[Figure]string{
		Pawn:   "P",
		Knight: "N",
		Bishop: "B",
		Rook:   "R",
		Queen:  "Q",
		King:   "K",
	}

	// pov xor mask indexed by color
	povMask = [ColorArraySize]Square{0x00, 0x38, 0x00}
)

// Color represents a side
type Color uint

const (
	NoColor Color = iota
	Black
	White

	ColorArraySize = int(iota)
	ColorMinValue  = Black
	ColorMaxValue  = White
)

// king home rank
var (
	kingHomeRank = [ColorArraySize]int{0, 7, 0}
)

// Square identifies the location on the board
type Square uint8

const (
	SquareA1 = Square(iota)
	SquareB1
	SquareC1
	SquareD1
	SquareE1
	SquareF1
	SquareG1
	SquareH1
	SquareA2
	SquareB2
	SquareC2
	SquareD2
	SquareE2
	SquareF2
	SquareG2
	SquareH2
	SquareA3
	SquareB3
	SquareC3
	SquareD3
	SquareE3
	SquareF3
	SquareG3
	SquareH3
	SquareA4
	SquareB4
	SquareC4
	SquareD4
	SquareE4
	SquareF4
	SquareG4
	SquareH4
	SquareA5
	SquareB5
	SquareC5
	SquareD5
	SquareE5
	SquareF5
	SquareG5
	SquareH5
	SquareA6
	SquareB6
	SquareC6
	SquareD6
	SquareE6
	SquareF6
	SquareG6
	SquareH6
	SquareA7
	SquareB7
	SquareC7
	SquareD7
	SquareE7
	SquareF7
	SquareG7
	SquareH7
	SquareA8
	SquareB8
	SquareC8
	SquareD8
	SquareE8
	SquareF8
	SquareG8
	SquareH8

	SquareArraySize = int(iota)
	SquareMinValue  = SquareA1
	SquareMaxValue  = SquareH8
)

// Bitboard is a set representing the 8x8 chess board squares
type Bitboard uint64

const (
	BbEmpty          Bitboard = 0x0000000000000000
	BbFull           Bitboard = 0xffffffffffffffff
	BbBorder         Bitboard = 0xff818181818181ff
	BbPawnStartRank  Bitboard = 0x00ff00000000ff00
	BbPawnDoubleRank Bitboard = 0x000000ffff000000
	BbBlackSquares   Bitboard = 0xaa55aa552a55aa55
	BbWhiteSquares   Bitboard = 0xd5aa55aad5aa55aa
)

const (
	BbFileA Bitboard = 0x101010101010101 << iota
	BbFileB
	BbFileC
	BbFileD
	BbFileE
	BbFileF
	BbFileG
	BbFileH
)

const (
	BbRank1 Bitboard = 0x0000000000000FF << (8 * iota)
	BbRank2
	BbRank3
	BbRank4
	BbRank5
	BbRank6
	BbRank7
	BbRank8
)

// Piece is a figure owned by one side
type Piece uint8

// Piece constants must stay in sync with ColorFigure
// the order of pieces must match Polyglot format:
// http://hgm.nubati.net/book_format.html
const (
	NoPiece Piece = iota
	_
	BlackPawn
	WhitePawn
	BlackKnight
	WhiteKnight
	BlackBishop
	WhiteBishop
	BlackRook
	WhiteRook
	BlackQueen
	WhiteQueen
	BlackKing
	WhiteKing

	PieceArraySize = int(iota)
	PieceMinValue  = BlackPawn
	PieceMaxValue  = WhiteKing
)

// debrujin constants
const (
	debrujinMul   = 0x218A392CD3D5DBF
	debrujinShift = 58
)

var debrujin64 = [64]uint{
	0, 1, 2, 7, 3, 13, 8, 19, 4, 25, 14, 28, 9, 34, 20, 40,
	5, 17, 26, 38, 15, 46, 29, 48, 10, 31, 35, 54, 21, 50, 41, 57,
	63, 6, 12, 18, 24, 27, 33, 39, 16, 37, 45, 47, 30, 53, 49, 56,
	62, 11, 23, 32, 36, 44, 52, 55, 61, 22, 43, 51, 60, 42, 59, 58,
}

// constants for popcnt
const (
	k1 = 0x5555555555555555
	k2 = 0x3333333333333333
	k4 = 0x0f0f0f0f0f0f0f0f
	kf = 0x0101010101010101
)

// MoveType defines the move type
type MoveType uint8

const (
	NoMove    MoveType = iota // no move or null move
	Normal                    // regular move
	Promotion                 // pawn is promoted. Move.Promotion() gives the new piece
	Castling                  // king castles
	Enpassant                 // pawn takes enpassant
)

const (
	// NullMove is a move that does nothing, has value to 0
	NullMove = Move(0)
)

// Move stores a position dependent move
//
// Bit representation
//   00.00.00.ff - from
//   00.00.ff.00 - to
//   00.0f.00.00 - move type
//   00.f0.00.00 - target
//   0f.00.00.00 - capture
//   f0.00.00.00 - piece
type Move uint32

// Castle represents the castling rights mask.
type Castle uint8

const (
	WhiteOO  Castle = 1 << iota // WhiteOO indicates that White can castle on King side.
	WhiteOOO                    // WhiteOOO indicates that White can castle on Queen side.
	BlackOO                     // BlackOO indicates that Black can castle on King side.
	BlackOOO                    // BlackOOO indicates that Black can castle on Queen side.

	NoCastle  Castle = 0                                       // NoCastle indicates no castling rights.
	AnyCastle Castle = WhiteOO | WhiteOOO | BlackOO | BlackOOO // AnyCastle indicates all castling rights.

	CastleArraySize = int(AnyCastle + 1)
	CastleMinValue  = NoCastle
	CastleMaxValue  = AnyCastle
)

var castleToString = [...]string{
	"-", "K", "Q", "KQ", "k", "Kk", "Qk", "KQk", "q", "Kq", "Qq", "KQq", "kq", "Kkq", "Qkq", "KQkq",
}

///////////////////////////////////////////////
// RankFile : returns a square with rank r and file f
// -> r int : rank , should be between 0 and 7
// -> f int : file , should be between 0 and 7
// <- Square : square

func RankFile(r, f int) Square {
	return Square(r*8 + f)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// SquareFromString : parses a square from a string
// -> s string : square in standard chess format [a-h][1-8]
// <- Square : square
// <- error : error

func SquareFromString(s string) (Square, error) {
	if len(s) != 2 {
		return SquareA1, fmt.Errorf("invalid square %s", s)
	}

	f, r := -1, -1
	if 'a' <= s[0] && s[0] <= 'h' {
		f = int(s[0] - 'a')
	}
	if 'A' <= s[0] && s[0] <= 'H' {
		f = int(s[0] - 'A')
	}
	if '1' <= s[1] && s[1] <= '8' {
		r = int(s[1] - '1')
	}
	if f == -1 || r == -1 {
		return SquareA1, fmt.Errorf("invalid square %s", s)
	}

	return RankFile(r, f), nil
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// Bitboard : returns a bitboard that has sq set
// -> sq : square
// <- Bitboard : bitboard

func (sq Square) Bitboard() Bitboard {
	return 1 << uint(sq)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// Relative : returns a square shifted with delta rank and delta file
// -> sq Square : square
// -> dr int : delta rank
// -> df int : delta file
// <- Square : square

func (sq Square) Relative(dr, df int) Square {
	return sq + Square(dr*8+df)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// Rank : returns a number from 0 to 7 representing the rank of the square
// -> sq Square : square
// <- int : rank

func (sq Square) Rank() int {
	return int(sq / 8)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// File : returns a number from 0 to 7 representing the file of the square
// -> sq Square : square
// <- int : file

func (sq Square) File() int {
	return int(sq % 8)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// POV : returns the square from col's point of view
// that is for Black the rank is flipped, file stays the same
// useful in evaluation based on king's or pawns' positions
// -> sq Square : square
// -> col Color : color
// <- Square : square

func (sq Square) POV(col Color) Square {
	return sq ^ povMask[col]
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// String : square in algebraic notation
// -> sq Square : square
// <- string : algeb

func (sq Square) String() string {
	return string([]byte{
		uint8(sq.File() + 'a'),
		uint8(sq.Rank() + '1'),
	})
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// Opposite : returns the reversed color of c
// -> c Color : color
// <- Color : reversed color , undefined if c is not White or Black

func (c Color) Opposite() Color {
	return White + Black - c
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// KingHomeRank : returns king's rank on starting position for color c
// -> c Color : color
// <- int : home rank , undefined if c is not White or Black

func (c Color) KingHomeRank() int {
	return kingHomeRank[c]
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// ColorFigure : returns a piece with col and fig
// -> col Color : color
// -> fig Figure : figure
// <- Piece : piece

func ColorFigure(col Color, fig Figure) Piece {
	return Piece(fig<<1) + Piece(col>>1)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// Color : returns piece's color
// -> pi : Piece
// <- Color : color

func (pi Piece) Color() Color {
	return Color(21844 >> pi & 3)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// Figure : returns piece's figure
// -> pi Piece : piece
// <- Figure: figure

func (pi Piece) Figure() Figure {
	return Figure(pi) >> 1
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// RankBb : returns a bitboard with all bits on rank set
// -> rank int : rank
// <- Bitboard : bitboard

func RankBb(rank int) Bitboard {
	return BbRank1 << uint(8*rank)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// FileBb : returns a bitboard with all bits on file set
// -> file int : file
// <- Bitboard : bitboard

func FileBb(file int) Bitboard {
	return BbFileA << uint(file)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// AdjacentFilesBb : returns a bitboard with all bits set on adjacent files
// -> file int : file
// <- Bitboard : bitboard

func AdjacentFilesBb(file int) Bitboard {
	var bb Bitboard
	if file > 0 {
		bb |= FileBb(file - 1)
	}
	if file < 7 {
		bb |= FileBb(file + 1)
	}
	return bb
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// North : shifts all squares one rank up
// -> bb Bitboard : bitboard
// <- Bitboard : bitboard shifted one rank up

func North(bb Bitboard) Bitboard {
	return bb << 8
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// South : shifts all squares one rank down
// -> bb Bitboard : bitboard
// <- Bitboard : bitboard shifted south

func South(bb Bitboard) Bitboard {
	return bb >> 8
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// East : shifts all squares one file right
// -> bb Bitboard : bitboard
// <- Bitboard : bitboard shifted east

func East(bb Bitboard) Bitboard {
	return bb &^ BbFileH << 1
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// West : shifts all squares one file left
// -> bb Bitboard : bitboard
// <- Bitboard : bitboard shifted west

func West(bb Bitboard) Bitboard {
	return bb &^ BbFileA >> 1
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// Forward : returns bb shifted one rank forward wrt color
// -> col Color : color
// -> bb Bitboard : bitboard
// <- Bitboard : bitboard shifted forward

func Forward(col Color, bb Bitboard) Bitboard {
	if col == White {
		return bb << 8
	}
	if col == Black {
		return bb >> 8
	}
	return bb
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// Backward : returns bb shifted one rank backward wrt color
// -> col Color : color
// -> bb Bitboard : bitboard
// <- Bitboard : bitboard shifted backward

func Backward(col Color, bb Bitboard) Bitboard {
	if col == White {
		return bb >> 8
	}
	if col == Black {
		return bb << 8
	}
	return bb
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// NorthFill : returns a bitboard with all north bits set
// -> bb Bitboard : bitboard
// <- Bitboard : bitboard north filled

func NorthFill(bb Bitboard) Bitboard {
	bb |= (bb << 8)
	bb |= (bb << 16)
	bb |= (bb << 24)
	return bb
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// SouthFill : returns a bitboard with all south bits set
// -> bb Bitboard : bitboard
// <- Bitboard : bitboard south filled

func SouthFill(bb Bitboard) Bitboard {
	bb |= (bb >> 8)
	bb |= (bb >> 16)
	bb |= (bb >> 24)
	return bb
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// Fill : returns a bitboard with all files with squares filled
// -> bb Bitboard : bitboard
// <- Bitboard : bitboard filled

func Fill(bb Bitboard) Bitboard {
	return NorthFill(bb) | SouthFill(bb)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// NorthSpan : is like NorthFill shifted on up
// -> bb Bitboard : bitboard
// <- Bitboard : bitboard north spanned

func NorthSpan(bb Bitboard) Bitboard {
	return NorthFill(North(bb))
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// SouthSpan : is like SouthFill shifted on up
// -> bb Bitboard : bitboard
// <- Bitboard : bitboard south spanned

func SouthSpan(bb Bitboard) Bitboard {
	return SouthFill(South(bb))
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// ForwardSpan : computes forward span wrt to color
// -> col Color : color
// -> bb Bitboard : bitboard
// <- bb Bitboard : bitboard forward spanned wrt color

func ForwardSpan(col Color, bb Bitboard) Bitboard {
	if col == White {
		return NorthSpan(bb)
	}
	if col == Black {
		return SouthSpan(bb)
	}
	return bb
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// BackwardSpan : computes backward span wrt to color
// -> col Color : color
// -> bb Bitboard : bitboard
// <- bb Bitboard : bitboard backward spanned

func BackwardSpan(col Color, bb Bitboard) Bitboard {
	if col == White {
		return SouthSpan(bb)
	}
	if col == Black {
		return NorthSpan(bb)
	}
	return bb
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// Has : returns true if sq is occupied in bitboard
// -> sq Square : square
// <- bool : true if sq is occupied

func (bb Bitboard) Has(sq Square) bool {
	return bb>>sq&1 != 0
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// AsSquare : returns the occupied square if the bitboard has a single piece
// -> bb Bitboard : bitboard
// <- Square : square , if the board has more then one piece the result is undefined

func (bb Bitboard) AsSquare() Square {
	// same as logN(bb)
	return Square(debrujin64[bb*debrujinMul>>debrujinShift])
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// LSB : picks a square in the board
// -> bb Bitboard : bitboard
// <- Bitboard : bitboard , empty board for empty board

func (bb Bitboard) LSB() Bitboard {
	return bb & (-bb)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// Count : counts number of squares set in bb
// same as popcnt
// -> bb Bitboard : bitboard
// <- int32 : number of bits set

func (bb Bitboard) Count() int32 {
	// code adapted from https://chessprogramming.wikispaces.com/Population+Count
	bb = bb - ((bb >> 1) & k1)
	bb = (bb & k2) + ((bb >> 2) & k2)
	bb = (bb + (bb >> 4)) & k4
	bb = (bb * kf) >> 56
	return int32(bb)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// CountMax2 : is equivalent to, but faster than, min(bb.Count(), 2)
// -> bb Bitboard : bitboard
// <- int32 : min(bb.Count(), 2)

func (bb Bitboard) CountMax2() int32 {
	if bb == 0 {
		return 0
	}
	if bb&(bb-1) == 0 {
		return 1
	}
	return 2
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// Pop : pops a set square from the bitboard
// -> bb *Bitboard : bitboard
// <- Square : square

func (bb *Bitboard) Pop() Square {
	sq := *bb & (-*bb)
	*bb -= sq
	// same as logN(sq)
	return Square(debrujin64[sq*debrujinMul>>debrujinShift])
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// MakeMove : constructs a move
// -> moveType MoveType : move type
// -> from Square : from square
// -> to Square : to square
// -> capture Piece : capture piece
// -> target Piece : target piece
// <- Move : move

func MakeMove(moveType MoveType, from, to Square, capture, target Piece) Move {
	piece := target
	if moveType == Promotion {
		piece = ColorFigure(target.Color(), Pawn)
	}

	return Move(from)<<0 +
		Move(to)<<8 +
		Move(moveType)<<16 +
		Move(target)<<20 +
		Move(capture)<<24 +
		Move(piece)<<28
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// From : returns the starting square of a move
// -> m Move : move
// <- Square: from square

func (m Move) From() Square {
	return Square(m >> 0 & 0xff)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// To : returns the destination square of a move
// -> m Move : move
// <- Square : to square

func (m Move) To() Square {
	return Square(m >> 8 & 0xff)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// MoveType : returns the move type of a move
// -> m Move : move
// <- MoveType : move type

func (m Move) MoveType() MoveType {
	return MoveType(m >> 16 & 0xf)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// Piece : returns the piece moved of a move
// -> m Move : move
// <- Pirece : piece moved

func (m Move) Piece() Piece {
	return Piece(m >> 28 & 0xf)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// SideToMove : returns which player is moving
// -> m Move : move
// <- Color : side to move

func (m Move) SideToMove() Color {
	return m.Piece().Color()
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// CaptureSquare : returns the captured piece square
// -> m Move : move
// <- Square : square , if no piece is captured, the result is the destination square

func (m Move) CaptureSquare() Square {
	if m.MoveType() != Enpassant {
		return m.To()
	}
	return m.From()&0x38 + m.To()&0x7
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// Capture : returns the captured piece
// -> m Move : move
// <- Piece : captured piece

func (m Move) Capture() Piece {
	return Piece(m >> 24 & 0xf)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// Target : returns the piece on the to square after the move is executed
// -> m Move : move
// <- Piece : piece on to square

func (m Move) Target() Piece {
	return Piece(m >> 20 & 0xf)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// Promotion : returns the promoted piece if any
// -> m Move : move
// <- Piece : promoted piece

func (m Move) Promotion() Piece {
	if m.MoveType() != Promotion {
		return NoPiece
	}
	return m.Target()
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// IsViolent : returns true if the move can change the position's score significantly
// TODO: IsViolent should be in sync with GenerateViolentMoves.
// -> m Move : move
// <- bool : true if move is violent

func (m Move) IsViolent() bool {
	return m.Capture() != NoPiece || m.MoveType() == Promotion
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// IsQuiet : returns true if the move is quiet
// -> m Move : move
// <- bool : true if move is quiet , in particular Castling is not quiet and not violent

func (m Move) IsQuiet() bool {
	return m.MoveType() == Normal && m.Capture() == NoPiece
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// UCI : converts a move to UCI format
// the protocol specification at http://wbec-ridderkerk.nl/html/UCIProtocol.html
// incorrectly states that this is the long algebraic notation (LAN)
// -> m Move : move
// <- string : uci move

func (m Move) UCI() string {
	return m.From().String() + m.To().String() + figureToSymbol[m.Promotion().Figure()]
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// LAN : converts a move to Long Algebraic Notation
// http://en.wikipedia.org/wiki/Algebraic_notation_%28chess%29#Long_algebraic_notation
// e.g. a2-a3, b7-b8Q, Nb1xc3
// -> m Move : move
// <- string : lan move

func (m Move) LAN() string {
	r := figureToSymbol[m.Piece().Figure()] + m.From().String()
	if m.Capture() != NoPiece {
		r += "x"
	} else {
		r += "-"
	}
	r += m.To().String() + figureToSymbol[m.Promotion().Figure()]
	return r
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// String : return move as string
// -> m Move : move
// <- string : move as string

func (m Move) String() string {
	return m.LAN()
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// String : castling rights as string
// -> c Castle : castling rights
// <- string : castling rights as string

func (c Castle) String() string {
	if c < NoCastle || c > AnyCastle {
		return fmt.Sprintf("Castle(%d)", c)
	}
	return castleToString[c]
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// CastlingRook : returns the rook moved during castling
// together with starting and stopping squares
// -> kingEnd Square : square
// <- Piece : castling rook
// <- Square : rook start square
// <- Square : rook end square

func CastlingRook(kingEnd Square) (Piece, Square, Square) {
	// explanation how rookStart works for king on E1
	// if kingEnd == C1 == b010, then rookStart == A1 == b000
	// if kingEnd == G1 == b110, then rookStart == H1 == b111
	// so bit 3 will set bit 2 and bit 1
	//
	// explanation how rookEnd works for king on E1
	// if kingEnd == C1 == b010, then rookEnd == D1 == b011
	// if kingEnd == G1 == b110, then rookEnd == F1 == b101
	// so bit 3 will invert bit 2. bit 1 is always set
	piece := Piece(Rook<<1) + (1 - Piece(kingEnd>>5))
	rookStart := kingEnd&^3 | (kingEnd & 4 >> 1) | (kingEnd & 4 >> 2)
	rookEnd := kingEnd ^ (kingEnd & 4 >> 1) | 1
	return piece, rookStart, rookEnd
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// logN : returns the logarithm of n, where n is a power of two
// -> n uint64 : n
// <- uint : log(n)

func logN(n uint64) uint {
	return debrujin64[n*debrujinMul>>debrujinShift]
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// popcnt : counts number of bits set in x
// -> x uint64 : x
// <- int : number of bits set

func popcnt(x uint64) int {
	// code adapted from https://chessprogramming.wikispaces.com/Population+Count
	x = x - ((x >> 1) & k1)
	x = (x & k2) + ((x >> 2) & k2)
	x = (x + (x >> 4)) & k4
	x = (x * kf) >> 56
	return int(x)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// max : returns maximum of a and b
// -> a int32 : a
// -> b int32 : b
// <- int32 : max(a,b)

func max(a, b int32) int32 {
	if a >= b {
		return a
	}
	return b
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// min : returns minimum of a and b
// -> a int32 : a
// -> b int32 : b
// <- int32 : min(a,b)

func min(a, b int32) int32 {
	if a <= b {
		return a
	}
	return b
}

///////////////////////////////////////////////
