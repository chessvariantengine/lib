//////////////////////////////////////////////////////
// movegen.go
// everything that is necessary for move generation
// zurichess sources: basic.go, misc.go, position.go, convert.go, polyglot.go, attack.go
//////////////////////////////////////////////////////

package lib

// imports

import(
	"fmt"
	"strconv"
	"math"
	"math/rand"
)

///////////////////////////////////////////////
// definitions

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

// figure to name
var FigureToName = [...]string{".","Pawn","Knight","Bishop","Rook","Queen","King"}

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

const (
	// Quiet indicates no capture, no castling, no promotion
	Quiet int = 1 << iota
	// Tactical indicates castling and underpromotions (including captures)
	Tactical
	// Violent indicates captures and queen promotions
	Violent
	// All indicates all moves
	All = Quiet | Tactical | Violent
)

var (
	// FENStartPos is the FEN string of the starting position
	FENStartPos = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

	// Which castle rights are lost when pieces are moved
	lostCastleRights [64]Castle
)

type explosion struct {
	sq Square    // square where the exploded piece was
	piece Piece  // exploded piece
}

type state struct {
	Zobrist         uint64    // Zobrist key
	Move            Move      // last move played
	HalfmoveClock   int       // last ply when a pawn was moved or a capture was made
	EnpassantSquare [2]Square // en passant square (polyglot, fen), if none, then SquareA1
	CastlingAbility Castle    // remaining castling rights
	ExplosionInfo   [8]explosion  // slice of exploded pieces ( max 8 )
	NumExplosions   int       // number of explosions
}

// Position represents the chess board and keeps track of the move history
type Position struct {
	ByFigure   [FigureArraySize]Bitboard // bitboards of square occupancy by figure
	ByColor    [ColorArraySize]Bitboard  // bitboards of square occupancy by color
	SideToMove Color                     // which side is to move. SideToMove is updated by DoMove and UndoMove
	Ply        int                       // current ply

	fullmoveCounter int     // fullmove counter, incremented after black move
	states          []state // a state for each Ply
	curr            *state  // current state
}

type castleInfo struct {
	Castle Castle
	Piece  [2]Piece
	Square [2]Square
}

var (
	itoa               = "0123456789" // shortcut for Itoa
	colorToSymbol      = "?bw"
	pieceToSymbol      = ".?pPnNbBrRqQkK"
	pieceToSymbolU     = []rune("☐?♙♟♘♞♗♝♖♜♕♛♔♚")
	
	symbolToCastleInfo = map[rune]castleInfo{
		'K': castleInfo{
			Castle: WhiteOO,
			Piece:  [2]Piece{WhiteKing, WhiteRook},
			Square: [2]Square{SquareE1, SquareH1},
		},
		'k': castleInfo{
			Castle: BlackOO,
			Piece:  [2]Piece{BlackKing, BlackRook},
			Square: [2]Square{SquareE8, SquareH8},
		},
		'Q': castleInfo{
			Castle: WhiteOOO,
			Piece:  [2]Piece{WhiteKing, WhiteRook},
			Square: [2]Square{SquareE1, SquareA1},
		},
		'q': castleInfo{
			Castle: BlackOOO,
			Piece:  [2]Piece{BlackKing, BlackRook},
			Square: [2]Square{SquareE8, SquareA8},
		},
	}
	symbolToColor = map[string]Color{
		"w": White,
		"b": Black,
	}
	symbolToPiece = map[rune]Piece{
		'p': BlackPawn,
		'n': BlackKnight,
		'b': BlackBishop,
		'r': BlackRook,
		'q': BlackQueen,
		'k': BlackKing,

		'P': WhitePawn,
		'N': WhiteKnight,
		'B': WhiteBishop,
		'R': WhiteRook,
		'Q': WhiteQueen,
		'K': WhiteKing,
	}
)

var (
	// the zobrist* arrays contain magic numbers used for Zobrist hashing
	zobristPiece     [PieceArraySize][SquareArraySize]uint64
	zobristEnpassant [SquareArraySize]uint64
	zobristCastle    [CastleArraySize]uint64
	zobristColor     [ColorArraySize]uint64

	// Polyglot random numbers
	// http://hgm.nubati.net/book_format.html
	// piece     (offset:   0, length: 768)
	// castle    (offset: 768, length:   4)
	// enpassant (offset: 772, length:   8)
	// turn      (offset: 780, length:   1)
	random64 = [781]uint64{
		0x9D39247E33776D41, 0x2AF7398005AAA5C7, 0x44DB015024623547, 0x9C15F73E62A76AE2,
		0x75834465489C0C89, 0x3290AC3A203001BF, 0x0FBBAD1F61042279, 0xE83A908FF2FB60CA,
		0x0D7E765D58755C10, 0x1A083822CEAFE02D, 0x9605D5F0E25EC3B0, 0xD021FF5CD13A2ED5,
		0x40BDF15D4A672E32, 0x011355146FD56395, 0x5DB4832046F3D9E5, 0x239F8B2D7FF719CC,
		0x05D1A1AE85B49AA1, 0x679F848F6E8FC971, 0x7449BBFF801FED0B, 0x7D11CDB1C3B7ADF0,
		0x82C7709E781EB7CC, 0xF3218F1C9510786C, 0x331478F3AF51BBE6, 0x4BB38DE5E7219443,
		0xAA649C6EBCFD50FC, 0x8DBD98A352AFD40B, 0x87D2074B81D79217, 0x19F3C751D3E92AE1,
		0xB4AB30F062B19ABF, 0x7B0500AC42047AC4, 0xC9452CA81A09D85D, 0x24AA6C514DA27500,
		0x4C9F34427501B447, 0x14A68FD73C910841, 0xA71B9B83461CBD93, 0x03488B95B0F1850F,
		0x637B2B34FF93C040, 0x09D1BC9A3DD90A94, 0x3575668334A1DD3B, 0x735E2B97A4C45A23,
		0x18727070F1BD400B, 0x1FCBACD259BF02E7, 0xD310A7C2CE9B6555, 0xBF983FE0FE5D8244,
		0x9F74D14F7454A824, 0x51EBDC4AB9BA3035, 0x5C82C505DB9AB0FA, 0xFCF7FE8A3430B241,
		0x3253A729B9BA3DDE, 0x8C74C368081B3075, 0xB9BC6C87167C33E7, 0x7EF48F2B83024E20,
		0x11D505D4C351BD7F, 0x6568FCA92C76A243, 0x4DE0B0F40F32A7B8, 0x96D693460CC37E5D,
		0x42E240CB63689F2F, 0x6D2BDCDAE2919661, 0x42880B0236E4D951, 0x5F0F4A5898171BB6,
		0x39F890F579F92F88, 0x93C5B5F47356388B, 0x63DC359D8D231B78, 0xEC16CA8AEA98AD76,
		0x5355F900C2A82DC7, 0x07FB9F855A997142, 0x5093417AA8A7ED5E, 0x7BCBC38DA25A7F3C,
		0x19FC8A768CF4B6D4, 0x637A7780DECFC0D9, 0x8249A47AEE0E41F7, 0x79AD695501E7D1E8,
		0x14ACBAF4777D5776, 0xF145B6BECCDEA195, 0xDABF2AC8201752FC, 0x24C3C94DF9C8D3F6,
		0xBB6E2924F03912EA, 0x0CE26C0B95C980D9, 0xA49CD132BFBF7CC4, 0xE99D662AF4243939,
		0x27E6AD7891165C3F, 0x8535F040B9744FF1, 0x54B3F4FA5F40D873, 0x72B12C32127FED2B,
		0xEE954D3C7B411F47, 0x9A85AC909A24EAA1, 0x70AC4CD9F04F21F5, 0xF9B89D3E99A075C2,
		0x87B3E2B2B5C907B1, 0xA366E5B8C54F48B8, 0xAE4A9346CC3F7CF2, 0x1920C04D47267BBD,
		0x87BF02C6B49E2AE9, 0x092237AC237F3859, 0xFF07F64EF8ED14D0, 0x8DE8DCA9F03CC54E,
		0x9C1633264DB49C89, 0xB3F22C3D0B0B38ED, 0x390E5FB44D01144B, 0x5BFEA5B4712768E9,
		0x1E1032911FA78984, 0x9A74ACB964E78CB3, 0x4F80F7A035DAFB04, 0x6304D09A0B3738C4,
		0x2171E64683023A08, 0x5B9B63EB9CEFF80C, 0x506AACF489889342, 0x1881AFC9A3A701D6,
		0x6503080440750644, 0xDFD395339CDBF4A7, 0xEF927DBCF00C20F2, 0x7B32F7D1E03680EC,
		0xB9FD7620E7316243, 0x05A7E8A57DB91B77, 0xB5889C6E15630A75, 0x4A750A09CE9573F7,
		0xCF464CEC899A2F8A, 0xF538639CE705B824, 0x3C79A0FF5580EF7F, 0xEDE6C87F8477609D,
		0x799E81F05BC93F31, 0x86536B8CF3428A8C, 0x97D7374C60087B73, 0xA246637CFF328532,
		0x043FCAE60CC0EBA0, 0x920E449535DD359E, 0x70EB093B15B290CC, 0x73A1921916591CBD,
		0x56436C9FE1A1AA8D, 0xEFAC4B70633B8F81, 0xBB215798D45DF7AF, 0x45F20042F24F1768,
		0x930F80F4E8EB7462, 0xFF6712FFCFD75EA1, 0xAE623FD67468AA70, 0xDD2C5BC84BC8D8FC,
		0x7EED120D54CF2DD9, 0x22FE545401165F1C, 0xC91800E98FB99929, 0x808BD68E6AC10365,
		0xDEC468145B7605F6, 0x1BEDE3A3AEF53302, 0x43539603D6C55602, 0xAA969B5C691CCB7A,
		0xA87832D392EFEE56, 0x65942C7B3C7E11AE, 0xDED2D633CAD004F6, 0x21F08570F420E565,
		0xB415938D7DA94E3C, 0x91B859E59ECB6350, 0x10CFF333E0ED804A, 0x28AED140BE0BB7DD,
		0xC5CC1D89724FA456, 0x5648F680F11A2741, 0x2D255069F0B7DAB3, 0x9BC5A38EF729ABD4,
		0xEF2F054308F6A2BC, 0xAF2042F5CC5C2858, 0x480412BAB7F5BE2A, 0xAEF3AF4A563DFE43,
		0x19AFE59AE451497F, 0x52593803DFF1E840, 0xF4F076E65F2CE6F0, 0x11379625747D5AF3,
		0xBCE5D2248682C115, 0x9DA4243DE836994F, 0x066F70B33FE09017, 0x4DC4DE189B671A1C,
		0x51039AB7712457C3, 0xC07A3F80C31FB4B4, 0xB46EE9C5E64A6E7C, 0xB3819A42ABE61C87,
		0x21A007933A522A20, 0x2DF16F761598AA4F, 0x763C4A1371B368FD, 0xF793C46702E086A0,
		0xD7288E012AEB8D31, 0xDE336A2A4BC1C44B, 0x0BF692B38D079F23, 0x2C604A7A177326B3,
		0x4850E73E03EB6064, 0xCFC447F1E53C8E1B, 0xB05CA3F564268D99, 0x9AE182C8BC9474E8,
		0xA4FC4BD4FC5558CA, 0xE755178D58FC4E76, 0x69B97DB1A4C03DFE, 0xF9B5B7C4ACC67C96,
		0xFC6A82D64B8655FB, 0x9C684CB6C4D24417, 0x8EC97D2917456ED0, 0x6703DF9D2924E97E,
		0xC547F57E42A7444E, 0x78E37644E7CAD29E, 0xFE9A44E9362F05FA, 0x08BD35CC38336615,
		0x9315E5EB3A129ACE, 0x94061B871E04DF75, 0xDF1D9F9D784BA010, 0x3BBA57B68871B59D,
		0xD2B7ADEEDED1F73F, 0xF7A255D83BC373F8, 0xD7F4F2448C0CEB81, 0xD95BE88CD210FFA7,
		0x336F52F8FF4728E7, 0xA74049DAC312AC71, 0xA2F61BB6E437FDB5, 0x4F2A5CB07F6A35B3,
		0x87D380BDA5BF7859, 0x16B9F7E06C453A21, 0x7BA2484C8A0FD54E, 0xF3A678CAD9A2E38C,
		0x39B0BF7DDE437BA2, 0xFCAF55C1BF8A4424, 0x18FCF680573FA594, 0x4C0563B89F495AC3,
		0x40E087931A00930D, 0x8CFFA9412EB642C1, 0x68CA39053261169F, 0x7A1EE967D27579E2,
		0x9D1D60E5076F5B6F, 0x3810E399B6F65BA2, 0x32095B6D4AB5F9B1, 0x35CAB62109DD038A,
		0xA90B24499FCFAFB1, 0x77A225A07CC2C6BD, 0x513E5E634C70E331, 0x4361C0CA3F692F12,
		0xD941ACA44B20A45B, 0x528F7C8602C5807B, 0x52AB92BEB9613989, 0x9D1DFA2EFC557F73,
		0x722FF175F572C348, 0x1D1260A51107FE97, 0x7A249A57EC0C9BA2, 0x04208FE9E8F7F2D6,
		0x5A110C6058B920A0, 0x0CD9A497658A5698, 0x56FD23C8F9715A4C, 0x284C847B9D887AAE,
		0x04FEABFBBDB619CB, 0x742E1E651C60BA83, 0x9A9632E65904AD3C, 0x881B82A13B51B9E2,
		0x506E6744CD974924, 0xB0183DB56FFC6A79, 0x0ED9B915C66ED37E, 0x5E11E86D5873D484,
		0xF678647E3519AC6E, 0x1B85D488D0F20CC5, 0xDAB9FE6525D89021, 0x0D151D86ADB73615,
		0xA865A54EDCC0F019, 0x93C42566AEF98FFB, 0x99E7AFEABE000731, 0x48CBFF086DDF285A,
		0x7F9B6AF1EBF78BAF, 0x58627E1A149BBA21, 0x2CD16E2ABD791E33, 0xD363EFF5F0977996,
		0x0CE2A38C344A6EED, 0x1A804AADB9CFA741, 0x907F30421D78C5DE, 0x501F65EDB3034D07,
		0x37624AE5A48FA6E9, 0x957BAF61700CFF4E, 0x3A6C27934E31188A, 0xD49503536ABCA345,
		0x088E049589C432E0, 0xF943AEE7FEBF21B8, 0x6C3B8E3E336139D3, 0x364F6FFA464EE52E,
		0xD60F6DCEDC314222, 0x56963B0DCA418FC0, 0x16F50EDF91E513AF, 0xEF1955914B609F93,
		0x565601C0364E3228, 0xECB53939887E8175, 0xBAC7A9A18531294B, 0xB344C470397BBA52,
		0x65D34954DAF3CEBD, 0xB4B81B3FA97511E2, 0xB422061193D6F6A7, 0x071582401C38434D,
		0x7A13F18BBEDC4FF5, 0xBC4097B116C524D2, 0x59B97885E2F2EA28, 0x99170A5DC3115544,
		0x6F423357E7C6A9F9, 0x325928EE6E6F8794, 0xD0E4366228B03343, 0x565C31F7DE89EA27,
		0x30F5611484119414, 0xD873DB391292ED4F, 0x7BD94E1D8E17DEBC, 0xC7D9F16864A76E94,
		0x947AE053EE56E63C, 0xC8C93882F9475F5F, 0x3A9BF55BA91F81CA, 0xD9A11FBB3D9808E4,
		0x0FD22063EDC29FCA, 0xB3F256D8ACA0B0B9, 0xB03031A8B4516E84, 0x35DD37D5871448AF,
		0xE9F6082B05542E4E, 0xEBFAFA33D7254B59, 0x9255ABB50D532280, 0xB9AB4CE57F2D34F3,
		0x693501D628297551, 0xC62C58F97DD949BF, 0xCD454F8F19C5126A, 0xBBE83F4ECC2BDECB,
		0xDC842B7E2819E230, 0xBA89142E007503B8, 0xA3BC941D0A5061CB, 0xE9F6760E32CD8021,
		0x09C7E552BC76492F, 0x852F54934DA55CC9, 0x8107FCCF064FCF56, 0x098954D51FFF6580,
		0x23B70EDB1955C4BF, 0xC330DE426430F69D, 0x4715ED43E8A45C0A, 0xA8D7E4DAB780A08D,
		0x0572B974F03CE0BB, 0xB57D2E985E1419C7, 0xE8D9ECBE2CF3D73F, 0x2FE4B17170E59750,
		0x11317BA87905E790, 0x7FBF21EC8A1F45EC, 0x1725CABFCB045B00, 0x964E915CD5E2B207,
		0x3E2B8BCBF016D66D, 0xBE7444E39328A0AC, 0xF85B2B4FBCDE44B7, 0x49353FEA39BA63B1,
		0x1DD01AAFCD53486A, 0x1FCA8A92FD719F85, 0xFC7C95D827357AFA, 0x18A6A990C8B35EBD,
		0xCCCB7005C6B9C28D, 0x3BDBB92C43B17F26, 0xAA70B5B4F89695A2, 0xE94C39A54A98307F,
		0xB7A0B174CFF6F36E, 0xD4DBA84729AF48AD, 0x2E18BC1AD9704A68, 0x2DE0966DAF2F8B1C,
		0xB9C11D5B1E43A07E, 0x64972D68DEE33360, 0x94628D38D0C20584, 0xDBC0D2B6AB90A559,
		0xD2733C4335C6A72F, 0x7E75D99D94A70F4D, 0x6CED1983376FA72B, 0x97FCAACBF030BC24,
		0x7B77497B32503B12, 0x8547EDDFB81CCB94, 0x79999CDFF70902CB, 0xCFFE1939438E9B24,
		0x829626E3892D95D7, 0x92FAE24291F2B3F1, 0x63E22C147B9C3403, 0xC678B6D860284A1C,
		0x5873888850659AE7, 0x0981DCD296A8736D, 0x9F65789A6509A440, 0x9FF38FED72E9052F,
		0xE479EE5B9930578C, 0xE7F28ECD2D49EECD, 0x56C074A581EA17FE, 0x5544F7D774B14AEF,
		0x7B3F0195FC6F290F, 0x12153635B2C0CF57, 0x7F5126DBBA5E0CA7, 0x7A76956C3EAFB413,
		0x3D5774A11D31AB39, 0x8A1B083821F40CB4, 0x7B4A38E32537DF62, 0x950113646D1D6E03,
		0x4DA8979A0041E8A9, 0x3BC36E078F7515D7, 0x5D0A12F27AD310D1, 0x7F9D1A2E1EBE1327,
		0xDA3A361B1C5157B1, 0xDCDD7D20903D0C25, 0x36833336D068F707, 0xCE68341F79893389,
		0xAB9090168DD05F34, 0x43954B3252DC25E5, 0xB438C2B67F98E5E9, 0x10DCD78E3851A492,
		0xDBC27AB5447822BF, 0x9B3CDB65F82CA382, 0xB67B7896167B4C84, 0xBFCED1B0048EAC50,
		0xA9119B60369FFEBD, 0x1FFF7AC80904BF45, 0xAC12FB171817EEE7, 0xAF08DA9177DDA93D,
		0x1B0CAB936E65C744, 0xB559EB1D04E5E932, 0xC37B45B3F8D6F2BA, 0xC3A9DC228CAAC9E9,
		0xF3B8B6675A6507FF, 0x9FC477DE4ED681DA, 0x67378D8ECCEF96CB, 0x6DD856D94D259236,
		0xA319CE15B0B4DB31, 0x073973751F12DD5E, 0x8A8E849EB32781A5, 0xE1925C71285279F5,
		0x74C04BF1790C0EFE, 0x4DDA48153C94938A, 0x9D266D6A1CC0542C, 0x7440FB816508C4FE,
		0x13328503DF48229F, 0xD6BF7BAEE43CAC40, 0x4838D65F6EF6748F, 0x1E152328F3318DEA,
		0x8F8419A348F296BF, 0x72C8834A5957B511, 0xD7A023A73260B45C, 0x94EBC8ABCFB56DAE,
		0x9FC10D0F989993E0, 0xDE68A2355B93CAE6, 0xA44CFE79AE538BBE, 0x9D1D84FCCE371425,
		0x51D2B1AB2DDFB636, 0x2FD7E4B9E72CD38C, 0x65CA5B96B7552210, 0xDD69A0D8AB3B546D,
		0x604D51B25FBF70E2, 0x73AA8A564FB7AC9E, 0x1A8C1E992B941148, 0xAAC40A2703D9BEA0,
		0x764DBEAE7FA4F3A6, 0x1E99B96E70A9BE8B, 0x2C5E9DEB57EF4743, 0x3A938FEE32D29981,
		0x26E6DB8FFDF5ADFE, 0x469356C504EC9F9D, 0xC8763C5B08D1908C, 0x3F6C6AF859D80055,
		0x7F7CC39420A3A545, 0x9BFB227EBDF4C5CE, 0x89039D79D6FC5C5C, 0x8FE88B57305E2AB6,
		0xA09E8C8C35AB96DE, 0xFA7E393983325753, 0xD6B6D0ECC617C699, 0xDFEA21EA9E7557E3,
		0xB67C1FA481680AF8, 0xCA1E3785A9E724E5, 0x1CFC8BED0D681639, 0xD18D8549D140CAEA,
		0x4ED0FE7E9DC91335, 0xE4DBF0634473F5D2, 0x1761F93A44D5AEFE, 0x53898E4C3910DA55,
		0x734DE8181F6EC39A, 0x2680B122BAA28D97, 0x298AF231C85BAFAB, 0x7983EED3740847D5,
		0x66C1A2A1A60CD889, 0x9E17E49642A3E4C1, 0xEDB454E7BADC0805, 0x50B704CAB602C329,
		0x4CC317FB9CDDD023, 0x66B4835D9EAFEA22, 0x219B97E26FFC81BD, 0x261E4E4C0A333A9D,
		0x1FE2CCA76517DB90, 0xD7504DFA8816EDBB, 0xB9571FA04DC089C8, 0x1DDC0325259B27DE,
		0xCF3F4688801EB9AA, 0xF4F5D05C10CAB243, 0x38B6525C21A42B0E, 0x36F60E2BA4FA6800,
		0xEB3593803173E0CE, 0x9C4CD6257C5A3603, 0xAF0C317D32ADAA8A, 0x258E5A80C7204C4B,
		0x8B889D624D44885D, 0xF4D14597E660F855, 0xD4347F66EC8941C3, 0xE699ED85B0DFB40D,
		0x2472F6207C2D0484, 0xC2A1E7B5B459AEB5, 0xAB4F6451CC1D45EC, 0x63767572AE3D6174,
		0xA59E0BD101731A28, 0x116D0016CB948F09, 0x2CF9C8CA052F6E9F, 0x0B090A7560A968E3,
		0xABEEDDB2DDE06FF1, 0x58EFC10B06A2068D, 0xC6E57A78FBD986E0, 0x2EAB8CA63CE802D7,
		0x14A195640116F336, 0x7C0828DD624EC390, 0xD74BBE77E6116AC7, 0x804456AF10F5FB53,
		0xEBE9EA2ADF4321C7, 0x03219A39EE587A30, 0x49787FEF17AF9924, 0xA1E9300CD8520548,
		0x5B45E522E4B1B4EF, 0xB49C3B3995091A36, 0xD4490AD526F14431, 0x12A8F216AF9418C2,
		0x001F837CC7350524, 0x1877B51E57A764D5, 0xA2853B80F17F58EE, 0x993E1DE72D36D310,
		0xB3598080CE64A656, 0x252F59CF0D9F04BB, 0xD23C8E176D113600, 0x1BDA0492E7E4586E,
		0x21E0BD5026C619BF, 0x3B097ADAF088F94E, 0x8D14DEDB30BE846E, 0xF95CFFA23AF5F6F4,
		0x3871700761B3F743, 0xCA672B91E9E4FA16, 0x64C8E531BFF53B55, 0x241260ED4AD1E87D,
		0x106C09B972D2E822, 0x7FBA195410E5CA30, 0x7884D9BC6CB569D8, 0x0647DFEDCD894A29,
		0x63573FF03E224774, 0x4FC8E9560F91B123, 0x1DB956E450275779, 0xB8D91274B9E9D4FB,
		0xA2EBEE47E2FBFCE1, 0xD9F1F30CCD97FB09, 0xEFED53D75FD64E6B, 0x2E6D02C36017F67F,
		0xA9AA4D20DB084E9B, 0xB64BE8D8B25396C1, 0x70CB6AF7C2D5BCF0, 0x98F076A4F7A2322E,
		0xBF84470805E69B5F, 0x94C3251F06F90CF3, 0x3E003E616A6591E9, 0xB925A6CD0421AFF3,
		0x61BDD1307C66E300, 0xBF8D5108E27E0D48, 0x240AB57A8B888B20, 0xFC87614BAF287E07,
		0xEF02CDD06FFDB432, 0xA1082C0466DF6C0A, 0x8215E577001332C8, 0xD39BB9C3A48DB6CF,
		0x2738259634305C14, 0x61CF4F94C97DF93D, 0x1B6BACA2AE4E125B, 0x758F450C88572E0B,
		0x959F587D507A8359, 0xB063E962E045F54D, 0x60E8ED72C0DFF5D1, 0x7B64978555326F9F,
		0xFD080D236DA814BA, 0x8C90FD9B083F4558, 0x106F72FE81E2C590, 0x7976033A39F7D952,
		0xA4EC0132764CA04B, 0x733EA705FAE4FA77, 0xB4D8F77BC3E56167, 0x9E21F4F903B33FD9,
		0x9D765E419FB69F6D, 0xD30C088BA61EA5EF, 0x5D94337FBFAF7F5B, 0x1A4E4822EB4D7A59,
		0x6FFE73E81B637FB3, 0xDDF957BC36D8B9CA, 0x64D0E29EEA8838B3, 0x08DD9BDFD96B9F63,
		0x087E79E5A57D1D13, 0xE328E230E3E2B3FB, 0x1C2559E30F0946BE, 0x720BF5F26F4D2EAA,
		0xB0774D261CC609DB, 0x443F64EC5A371195, 0x4112CF68649A260E, 0xD813F2FAB7F5C5CA,
		0x660D3257380841EE, 0x59AC2C7873F910A3, 0xE846963877671A17, 0x93B633ABFA3469F8,
		0xC0C0F5A60EF4CDCF, 0xCAF21ECD4377B28C, 0x57277707199B8175, 0x506C11B9D90E8B1D,
		0xD83CC2687A19255F, 0x4A29C6465A314CD1, 0xED2DF21216235097, 0xB5635C95FF7296E2,
		0x22AF003AB672E811, 0x52E762596BF68235, 0x9AEBA33AC6ECC6B0, 0x944F6DE09134DFB6,
		0x6C47BEC883A7DE39, 0x6AD047C430A12104, 0xA5B1CFDBA0AB4067, 0x7C45D833AFF07862,
		0x5092EF950A16DA0B, 0x9338E69C052B8E7B, 0x455A4B4CFE30E3F5, 0x6B02E63195AD0CF8,
		0x6B17B224BAD6BF27, 0xD1E0CCD25BB9C169, 0xDE0C89A556B9AE70, 0x50065E535A213CF6,
		0x9C1169FA2777B874, 0x78EDEFD694AF1EED, 0x6DC93D9526A50E68, 0xEE97F453F06791ED,
		0x32AB0EDB696703D3, 0x3A6853C7E70757A7, 0x31865CED6120F37D, 0x67FEF95D92607890,
		0x1F2B1D1F15F6DC9C, 0xB69E38A8965C6B65, 0xAA9119FF184CCCF4, 0xF43C732873F24C13,
		0xFB4A3D794A9A80D2, 0x3550C2321FD6109C, 0x371F77E76BB8417E, 0x6BFA9AAE5EC05779,
		0xCD04F3FF001A4778, 0xE3273522064480CA, 0x9F91508BFFCFC14A, 0x049A7F41061A9E60,
		0xFCB6BE43A9F2FE9B, 0x08DE8A1C7797DA9B, 0x8F9887E6078735A1, 0xB5B4071DBFC73A66,
		0x230E343DFBA08D33, 0x43ED7F5A0FAE657D, 0x3A88A0FBBCB05C63, 0x21874B8B4D2DBC4F,
		0x1BDEA12E35F6A8C9, 0x53C065C6C8E63528, 0xE34A1D250E7A8D6B, 0xD6B04D3B7651DD7E,
		0x5E90277E7CB39E2D, 0x2C046F22062DC67D, 0xB10BB459132D0A26, 0x3FA9DDFB67E2F199,
		0x0E09B88E1914F7AF, 0x10E8B35AF3EEAB37, 0x9EEDECA8E272B933, 0xD4C718BC4AE8AE5F,
		0x81536D601170FC20, 0x91B534F885818A06, 0xEC8177F83F900978, 0x190E714FADA5156E,
		0xB592BF39B0364963, 0x89C350C893AE7DC1, 0xAC042E70F8B383F2, 0xB49B52E587A1EE60,
		0xFB152FE3FF26DA89, 0x3E666E6F69AE2C15, 0x3B544EBE544C19F9, 0xE805A1E290CF2456,
		0x24B33C9D7ED25117, 0xE74733427B72F0C1, 0x0A804D18B7097475, 0x57E3306D881EDB4F,
		0x4AE7D6A36EB5DBCB, 0x2D8D5432157064C8, 0xD1E649DE1E7F268B, 0x8A328A1CEDFE552C,
		0x07A3AEC79624C7DA, 0x84547DDC3E203C94, 0x990A98FD5071D263, 0x1A4FF12616EEFC89,
		0xF6F7FD1431714200, 0x30C05B1BA332F41C, 0x8D2636B81555A786, 0x46C9FEB55D120902,
		0xCCEC0A73B49C9921, 0x4E9D2827355FC492, 0x19EBB029435DCB0F, 0x4659D2B743848A2C,
		0x963EF2C96B33BE31, 0x74F85198B05A2E7D, 0x5A0F544DD2B1FB18, 0x03727073C2E134B1,
		0xC7F6AA2DE59AEA61, 0x352787BAA0D7C22F, 0x9853EAB63B5E0B35, 0xABBDCDD7ED5C0860,
		0xCF05DAF5AC8D77B0, 0x49CAD48CEBF4A71E, 0x7A4C10EC2158C4A6, 0xD9E92AA246BF719E,
		0x13AE978D09FE5557, 0x730499AF921549FF, 0x4E4B705B92903BA4, 0xFF577222C14F0A3A,
		0x55B6344CF97AAFAE, 0xB862225B055B6960, 0xCAC09AFBDDD2CDB4, 0xDAF8E9829FE96B5F,
		0xB5FDFC5D3132C498, 0x310CB380DB6F7503, 0xE87FBB46217A360E, 0x2102AE466EBB1148,
		0xF8549E1A3AA5E00D, 0x07A69AFDCC42261A, 0xC4C118BFE78FEAAE, 0xF9F4892ED96BD438,
		0x1AF3DBE25D8F45DA, 0xF5B4B0B0D2DEEEB4, 0x962ACEEFA82E1C84, 0x046E3ECAAF453CE9,
		0xF05D129681949A4C, 0x964781CE734B3C84, 0x9C2ED44081CE5FBD, 0x522E23F3925E319E,
		0x177E00F9FC32F791, 0x2BC60A63A6F3B3F2, 0x222BBFAE61725606, 0x486289DDCC3D6780,
		0x7DC7785B8EFDFC80, 0x8AF38731C02BA980, 0x1FAB64EA29A2DDF7, 0xE4D9429322CD065A,
		0x9DA058C67844F20C, 0x24C0E332B70019B0, 0x233003B5A6CFE6AD, 0xD586BD01C5C217F6,
		0x5E5637885F29BC2B, 0x7EBA726D8C94094B, 0x0A56A5F0BFE39272, 0xD79476A84EE20D06,
		0x9E4C1269BAA4BF37, 0x17EFEE45B0DEE640, 0x1D95B0A5FCF90BC6, 0x93CBE0B699C2585D,
		0x65FA4F227A2B6D79, 0xD5F9E858292504D5, 0xC2B5A03F71471A6F, 0x59300222B4561E00,
		0xCE2F8642CA0712DC, 0x7CA9723FBB2E8988, 0x2785338347F2BA08, 0xC61BB3A141E50E8C,
		0x150F361DAB9DEC26, 0x9F6A419D382595F4, 0x64A53DC924FE7AC9, 0x142DE49FFF7A7C3D,
		0x0C335248857FA9E7, 0x0A9C32D5EAE45305, 0xE6C42178C4BBB92E, 0x71F1CE2490D20B07,
		0xF1BCC3D275AFE51A, 0xE728E8C83C334074, 0x96FBF83A12884624, 0x81A1549FD6573DA5,
		0x5FA7867CAF35E149, 0x56986E2EF3ED091B, 0x917F1DD5F8886C61, 0xD20D8C88C8FFE65F,
		0x31D71DCE64B2C310, 0xF165B587DF898190, 0xA57E6339DD2CF3A0, 0x1EF6E6DBB1961EC9,
		0x70CC73D90BC26E24, 0xE21A6B35DF0C3AD7, 0x003A93D8B2806962, 0x1C99DED33CB890A1,
		0xCF3145DE0ADD4289, 0xD0E4427A5514FB72, 0x77C621CC9FB3A483, 0x67A34DAC4356550B,
		0xF8D626AAAF278509,
	}
)

var (
	// bbPawnAttack contains pawn's attack tables.
	bbPawnAttack [64]Bitboard
	// bbKnightAttack contains knight's attack tables.
	bbKnightAttack [64]Bitboard
	// bbKingAttack contains king's attack tables (excluding castling).
	bbKingAttack [64]Bitboard
	// bbSuperAttack contains queen piece's attack tables. This queen can jump.
	bbSuperAttack [64]Bitboard

	rookMagic    [64]magicInfo
	rookDeltas   = [][2]int{{-1, +0}, {+1, +0}, {+0, -1}, {+0, +1}}
	bishopMagic  [64]magicInfo
	bishopDeltas = [][2]int{{-1, +1}, {+1, +1}, {+1, -1}, {-1, -1}}
)

type wizard struct {
	// Sliding deltas.
	Deltas        [][2]int
	MinShift      uint // Which shifts to search.
	MaxShift      uint
	MaxNumEntries uint // How much to search.
	Rand          *rand.Rand

	numMagicTests uint
	magics        [64]uint64
	shifts        [64]uint // Number of bits for indexes.

	store     []Bitboard // Temporary store to check hash collisions.
	reference []Bitboard
	occupancy []Bitboard
}

type magicInfo struct {
	store []Bitboard // attack boards of size 1<<(64-shift)
	mask  Bitboard   // square's mask.
	magic uint64     // magic multiplier
	shift uint       // shift bits to index store
	pad   [2]uint64  // padding so the structure has 32 bytes.
}

const GET_FIRST              = true
const GET_ALL                = false

var (
	errorWrongLength       = fmt.Errorf("SAN string is too short")
	errorUnknownFigure     = fmt.Errorf("unknown figure symbol")
	errorBadDisambiguation = fmt.Errorf("bad disambiguation")
	errorBadPromotion      = fmt.Errorf("only pawns on the last rank can be promoted")
	errorNoSuchMove        = fmt.Errorf("no such move")

	// maps runes to figures
	symbolToFigure = map[rune]Figure{
		'p': Pawn,
		'n': Knight,
		'b': Bishop,
		'r': Rook,
		'q': Queen,
		'k': King,

		'P': Pawn,
		'N': Knight,
		'B': Bishop,
		'R': Rook,
		'Q': Queen,
		'K': King,
	}
)

// list of squares
type squarelist []Square

// keeps track of which squares can be exploded around a square
// can be used to establish adjacent kings also
var explosionsquares [SquareArraySize]squarelist

var explosionbitboards [SquareArraySize]Bitboard

// end definitions
///////////////////////////////////////////////

///////////////////////////////////////////////
// init : initialization

func init() {
	// lost castle rights
	lostCastleRights[SquareA1] = WhiteOOO
	lostCastleRights[SquareE1] = WhiteOOO | WhiteOO
	lostCastleRights[SquareH1] = WhiteOO
	lostCastleRights[SquareA8] = BlackOOO
	lostCastleRights[SquareE8] = BlackOOO | BlackOO
	lostCastleRights[SquareH8] = BlackOO

	// attacks
	initBbPawnAttack()
	initBbKnightAttack()
	initBbKingAttack()
	initBbSuperAttack()
	initRookMagic()
	initBishopMagic()

	// init Zobrist
	initZobristPiece()
	initZobristEnpassant()
	initZobristCastle()
	initZobristColor()

	// init explosionsquares
	for sq := SquareMinValue ; sq <=SquareMaxValue ; sq ++ {
		var bb Bitboard = 0
		squares := squarelist{}
		r := sq.Rank()
		f := sq.File()
		for dr := -1 ; dr <= 1 ; dr+=1 {
			for df := -1 ; df <= 1; df+=1 {
				er := r + dr
				ef := f + df
				if ( ( dr != 0 ) || ( df != 0 ) ) && RankFileOk(er, ef) {
					esq := RankFile(er, ef)
					squares = append(squares, esq)
					sbb := esq.Bitboard()
					bb = bb | sbb
				}
			}
		}
		explosionsquares[sq]=squares
		explosionbitboards[sq]=bb
	}
	/*for sq := SquareMinValue ; sq <=SquareMaxValue ; sq ++ {
		explosionbitboards[sq].Print()
	}*/
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// GetKingBitboard : get the king bitboard for side
// -> pos *Position : position
// -> side Color : side
// <- Bitboard : king bitboard for side

func (pos *Position) GetKingBitboard(side Color) Bitboard {
	return pos.ByColor[side] & pos.ByFigure[King]
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// KingsAdjacent : determine if kings are on adjacent squares
// -> pos *Position : position
// <- bool : true if kings are adjacent

func (pos *Position) KingsAdjacent() bool {
	wkbb := pos.GetKingBitboard(White)
	if wkbb == 0 {
		// white king missing, cannot be adjacent
		return false
	}
	bkbb := pos.GetKingBitboard(Black)
	if bkbb == 0 {
		// black king missing, cannot be adjacent
		return false
	}
	// get white king's square
	wksq := wkbb.Pop()
	// bitboard for neighbours of white king
	neighboursbb := explosionbitboards[wksq]
	if (neighboursbb & bkbb) == 0 {
		return false
	}
	return true
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// RankOk : check if rank is within the board
// -> r int : rank
// <- bool : true if rank is within the board

func RankOk(r int) bool {
	return (r>=0) && (r<=7)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// FileOk : check if file is within the board
// -> f int : file
// <- bool : true if file is within the board

func FileOk(f int) bool {
	return (f>=0) && (f<=7)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// RankFileOk : check if rank and file are within the board
// -> f int : file
// -> r int : rank
// <- bool : true if file and rank are both within the board

func RankFileOk(r, f int) bool {
	return RankOk(r) && FileOk(f)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// initZobristPiece : init Zobrist piece

func initZobristPiece() {
	for pi := PieceMinValue; pi <= PieceMaxValue; pi++ {
		for sq := SquareMinValue; sq <= SquareMaxValue; sq++ {
			i := int(pi-PieceMinValue)*64 + int(sq)
			zobristPiece[pi][sq] = random64[i]
		}
	}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// initZobristEnpassant : init Zobrist en passant

func initZobristEnpassant() {
	for i := 0; i < 8; i++ {
		zobristEnpassant[SquareA3+Square(i)] = random64[772+i]
		zobristEnpassant[SquareA6+Square(i)] = random64[772+i]
	}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// initZobristCastle : init Zobrist castle

func initZobristCastle() {
	for i := CastleMinValue; i <= CastleMaxValue; i++ {
		if i&WhiteOO != 0 {
			zobristCastle[i] ^= random64[768]
		}
		if i&WhiteOOO != 0 {
			zobristCastle[i] ^= random64[769]
		}
		if i&BlackOO != 0 {
			zobristCastle[i] ^= random64[770]
		}
		if i&BlackOOO != 0 {
			zobristCastle[i] ^= random64[771]
		}
	}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// initZobristColor : init Zobrist color

func initZobristColor() {
	zobristColor[White] = random64[780]
}

///////////////////////////////////////////////

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

///////////////////////////////////////////////
// initJumpAttack : init jump attacks
// -> jump [][2]int : jump
// -> attack []Bitboard : attack

func initJumpAttack(jump [][2]int, attack []Bitboard) {
	for r := 0; r < 8; r++ {
		for f := 0; f < 8; f++ {
			bb := Bitboard(0)
			for _, d := range jump {
				r0, f0 := r+d[0], f+d[1]
				if 0 > r0 || r0 >= 8 || 0 > f0 || f0 >= 8 {
					continue
				}
				bb |= RankFile(r0, f0).Bitboard()
			}
			attack[RankFile(r, f)] = bb
		}
	}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// initBbPawnAttack : init pawn attacks

func initBbPawnAttack() {
	pawnJump := [][2]int{
		{-1, -1}, {-1, +1}, {+1, +1}, {+1, -1},
	}
	initJumpAttack(pawnJump, bbPawnAttack[:])
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// initBbKnightAttack : init knight attacks

func initBbKnightAttack() {
	knightJump := [][2]int{
		{-2, -1}, {-2, +1}, {+2, -1}, {+2, +1},
		{-1, -2}, {-1, +2}, {+1, -2}, {+1, +2},
	}
	initJumpAttack(knightJump, bbKnightAttack[:])
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// initBbKingAttack : init king attacks

func initBbKingAttack() {
	kingJump := [][2]int{
		{-1, -1}, {-1, +0}, {-1, +1}, {+0, +1},
		{+1, +1}, {+1, +0}, {+1, -1}, {+0, -1},
	}
	initJumpAttack(kingJump, bbKingAttack[:])
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// slidingAttack : sliding attacks
// -> sq Square : square
// -> deltas [][2]int : deltas
// -> occupancy Bitboard : occupancy
// <- Bitboard : attacks bitboard

func slidingAttack(sq Square, deltas [][2]int, occupancy Bitboard) Bitboard {
	r, f := sq.Rank(), sq.File()
	bb := Bitboard(0)
	for _, d := range deltas {
		r0, f0 := r, f
		for {
			r0, f0 = r0+d[0], f0+d[1]
			if 0 > r0 || r0 >= 8 || 0 > f0 || f0 >= 8 {
				// Stop when outside of the board.
				break
			}
			sq0 := RankFile(r0, f0)
			bb |= sq0.Bitboard()
			if occupancy&sq0.Bitboard() != 0 {
				// Stop when a piece was hit.
				break
			}
		}
	}
	return bb
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// SetMagic : set magic
// -> mi []magicInfo : magic info
// -> sq Square : squareű
// -> magic uint64 : magic
// -> shift uint : shift

func (wiz *wizard) SetMagic(mi []magicInfo, sq Square, magic uint64, shift uint) {
	wiz.prepare(sq)
	if !wiz.tryMagicNumber(&mi[sq], sq, magic, shift) {
		panic(fmt.Sprintf("invalid magic: sq=%v magic=%d shift=%d", sq, magic, shift))
	}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// prepare : computes reference and occupancy tables for a square
// -> wiz *wizard : wizard
// -> sq Square : square

func (wiz *wizard) prepare(sq Square) {
	wiz.reference = wiz.reference[:0]
	wiz.occupancy = wiz.occupancy[:0]

	// Carry-Rippler trick to enumerate all subsets of mask.
	for mask, subset := wiz.mask(sq), Bitboard(0); ; {
		attack := slidingAttack(sq, wiz.Deltas, subset)
		wiz.reference = append(wiz.reference, subset)
		wiz.occupancy = append(wiz.occupancy, attack)
		subset = (subset - mask) & mask
		if subset == 0 {
			break
		}
	}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// tryMagicNumber : try magic number
// -> wiz *wizard : wizard
// -> mi *magicInfo : magic info
// -> sq Square : square
// -> magic uint64 : magic
// -> shift uint : shift
// -> bool : true if ok

func (wiz *wizard) tryMagicNumber(mi *magicInfo, sq Square, magic uint64, shift uint) bool {
	wiz.numMagicTests++

	// Clear store.
	if len(wiz.store) < 1<<shift {
		wiz.store = make([]Bitboard, 1<<shift)
	}
	for j := range wiz.store[:1<<shift] {
		wiz.store[j] = 0
	}

	// Verify that magic gives a perfect hash.
	for i, bb := range wiz.reference {
		index := spell(magic, 32-shift, bb)
		if wiz.store[index] != 0 && wiz.store[index] != wiz.occupancy[i] {
			return false
		}
		wiz.store[index] = wiz.occupancy[i]
	}

	// Perfect hash, store it.
	wiz.magics[sq] = magic
	wiz.shifts[sq] = shift

	mi.store = make([]Bitboard, 1<<shift)
	copy(mi.store, wiz.store)
	mi.mask = wiz.mask(sq)
	mi.magic = magic
	mi.shift = 32 - shift
	return true
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// mask : returns the attack set on empty board minus the border
// -> wiz *wizard : wizard
// -> sq Square : square
// <- Bitboard : attack bitboard

func (wiz *wizard) mask(sq Square) Bitboard {
	// Compute border. Trick source: stockfish.
	border := (BbRank1 | BbRank8) & ^RankBb(sq.Rank())
	border |= (BbFileA | BbFileH) & ^FileBb(sq.File())
	return ^border & slidingAttack(sq, wiz.Deltas, BbEmpty)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// spell : spell
// magic uint64 : magic
// shift uint : shift
// bb Bitboard : Bitboard
// <- uint : spell

func spell(magic uint64, shift uint, bb Bitboard) uint {
	mul := magic * uint64(bb)
	return uint(uint32(mul>>32^mul) >> shift)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// searchMagic : search magic
// -> wiz *wizard : wizard
// -> sq Square : square
// -> mi *magicInfo : magic info

func (wiz *wizard) searchMagic(sq Square, mi *magicInfo) {
	if wiz.shifts[sq] != 0 && wiz.shifts[sq] <= wiz.MinShift {
		// Don't search if shift is low enough.
		return
	}

	// Try magic numbers with small shifts.
	wiz.prepare(sq)
	mask := wiz.mask(sq)
	for i := 0; i < 100 || wiz.shifts[sq] == 0; i++ {
		// Pick a smaller shift than current best.
		var shift uint
		if wiz.shifts[sq] == 0 {
			shift = wiz.MaxShift
		} else {
			shift = wiz.shifts[sq] - 1
		}

		// Pick a good magic and test whether it gives a perfect hash.
		var magic uint64
		for popcnt(uint64(mask)*magic) < 6 {
			magic = wiz.randMagic()
		}
		wiz.tryMagicNumber(mi, sq, magic, shift)
	}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// SearchMagic : finds new magics
// -> wiz *wizard : wizard
// -> mi []magicInfo : magic info

func (wiz *wizard) SearchMagics(mi []magicInfo) {
	numEntries := uint(math.MaxUint32)
	minShift := uint(math.MaxUint32)
	for numEntries > wiz.MaxNumEntries {
		numEntries = 0
		for sq := SquareMinValue; sq <= SquareMaxValue; sq++ {
			wiz.searchMagic(sq, &mi[sq])
			numEntries += 1 << wiz.shifts[sq]
			if minShift > wiz.shifts[sq] {
				minShift = wiz.shifts[sq]
			}
		}
	}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// randMagic : returns a random magic number
// -> wiz *wizard : wizard
// <- uint64 : random magic

func (wiz *wizard) randMagic() uint64 {
	r := uint64(wiz.Rand.Int63())
	r &= uint64(wiz.Rand.Int63())
	r &= uint64(wiz.Rand.Int63())
	return r<<6 + 1
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// KnightMobility : returns all squares a knight can reach from sq
// -> sq Square : square
// <- Bitboard : reachable squares

func KnightMobility(sq Square) Bitboard {
	return bbKnightAttack[sq]
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// BishopMobility : returns the squares a bishop can reach from sq given all pieces
// -> sq Square : square
// -> all Bitboard : all pieces
// <- Bitboard : reachable squares

func BishopMobility(sq Square, all Bitboard) Bitboard {
	return bishopMagic[sq].Attack(all)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// RookMobility : returns the squares a rook can reach from sq given all pieces
// -> sq Square : square
// -> all Bitboard : all pieces
// <- Bitboard : reachable squares

func RookMobility(sq Square, all Bitboard) Bitboard {
	return rookMagic[sq].Attack(all)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// QueenMobility : returns the squares a queen can reach from sq given all pieces
// -> sq Square : square
// -> all Bitboard : all pieces
// <- Bitboard : reachable squares

func QueenMobility(sq Square, all Bitboard) Bitboard {
	return rookMagic[sq].Attack(all) | bishopMagic[sq].Attack(all)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// KingMobility : returns all squares a king can reach from sq
// doesn't include castling
// -> sq Square : square
// -> all Bitboard : all pieces
// <- Bitboard : reachable squares

func KingMobility(sq Square) Bitboard {
	return bbKingAttack[sq]
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// Attack : attack
// -> mi *magicInfo : magic info
// -> ref Bitboard : ref
// <- Bitboard : attack bitboard

func (mi *magicInfo) Attack(ref Bitboard) Bitboard {
	return mi.store[spell(mi.magic, mi.shift, ref&mi.mask)]
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// initBishopMagic : init bishop magic

func initBishopMagic() {
	wiz := &wizard{
		Deltas:        bishopDeltas,
		MinShift:      5,
		MaxShift:      9,
		MaxNumEntries: 6000,
		Rand:          rand.New(rand.NewSource(1)),
	}

	// Bishop magics, unlike rook magics are easy to find.
	wiz.SearchMagics(bishopMagic[:])
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// initRookMagic : init rook magic

func initRookMagic() {
	wiz := &wizard{
		Deltas:        rookDeltas,
		MinShift:      10,
		MaxShift:      13,
		MaxNumEntries: 130000,
		Rand:          rand.New(rand.NewSource(1)),
	}

	// a set of known good magics for rook
	// finding good rook magics is slow, so we just use some precomputed values
	// for readability reasons, do not make an array
	wiz.SetMagic(rookMagic[:], SquareA1, 36028952711532673, 12)
	wiz.SetMagic(rookMagic[:], SquareA2, 5066692388487169, 11)
	wiz.SetMagic(rookMagic[:], SquareA3, 4631389266822304769, 11)
	wiz.SetMagic(rookMagic[:], SquareA4, 10450310413697025, 11)
	wiz.SetMagic(rookMagic[:], SquareA5, 140737496752193, 11)
	wiz.SetMagic(rookMagic[:], SquareA6, 4755801345016995841, 11)
	wiz.SetMagic(rookMagic[:], SquareA7, 2310346608845258881, 11)
	wiz.SetMagic(rookMagic[:], SquareA8, 1153273486052196353, 12)
	wiz.SetMagic(rookMagic[:], SquareB1, 14411536674683101313, 11)
	wiz.SetMagic(rookMagic[:], SquareB2, 360288245069774977, 10)
	wiz.SetMagic(rookMagic[:], SquareB3, 9304436831221219585, 10)
	wiz.SetMagic(rookMagic[:], SquareB4, 90107726679507201, 10)
	wiz.SetMagic(rookMagic[:], SquareB5, 23081233739161857, 10)
	wiz.SetMagic(rookMagic[:], SquareB6, 17610976739329, 10)
	wiz.SetMagic(rookMagic[:], SquareB7, 9007201406419201, 10)
	wiz.SetMagic(rookMagic[:], SquareB8, 846729215754241, 11)
	wiz.SetMagic(rookMagic[:], SquareC1, 576496005395513857, 11)
	wiz.SetMagic(rookMagic[:], SquareC2, 2355383154875302401, 10)
	wiz.SetMagic(rookMagic[:], SquareC3, 9263904435128516865, 10)
	wiz.SetMagic(rookMagic[:], SquareC4, 9223653580555165697, 10)
	wiz.SetMagic(rookMagic[:], SquareC5, 216208542045048897, 10)
	wiz.SetMagic(rookMagic[:], SquareC6, 2667820173397917761, 10)
	wiz.SetMagic(rookMagic[:], SquareC7, 360428707682197761, 10)
	wiz.SetMagic(rookMagic[:], SquareC8, 4611695089401765889, 11)
	wiz.SetMagic(rookMagic[:], SquareD1, 4604372721729, 11)
	wiz.SetMagic(rookMagic[:], SquareD2, 9304436898871644161, 10)
	wiz.SetMagic(rookMagic[:], SquareD3, 596726951168704769, 10)
	wiz.SetMagic(rookMagic[:], SquareD4, 5190691178076966913, 10)
	wiz.SetMagic(rookMagic[:], SquareD5, 4655469687738433, 10)
	wiz.SetMagic(rookMagic[:], SquareD6, 5764660368316567553, 10)
	wiz.SetMagic(rookMagic[:], SquareD7, 2452350872031592705, 10)
	wiz.SetMagic(rookMagic[:], SquareD8, 1153211792858550273, 11)
	wiz.SetMagic(rookMagic[:], SquareE1, 36031546200687617, 11)
	wiz.SetMagic(rookMagic[:], SquareE2, 144115499663886337, 10)
	wiz.SetMagic(rookMagic[:], SquareE3, 288388705826635841, 10)
	wiz.SetMagic(rookMagic[:], SquareE4, 74380329532524545, 10)
	wiz.SetMagic(rookMagic[:], SquareE5, 4910190248417298433, 10)
	wiz.SetMagic(rookMagic[:], SquareE6, 2251851487527425, 10)
	wiz.SetMagic(rookMagic[:], SquareE7, 7881299415531649, 10)
	wiz.SetMagic(rookMagic[:], SquareE8, 54342271281408001, 11)
	wiz.SetMagic(rookMagic[:], SquareF1, 36033197213089793, 11)
	wiz.SetMagic(rookMagic[:], SquareF2, 108086941350626369, 10)
	wiz.SetMagic(rookMagic[:], SquareF3, 1298162592589676609, 10)
	wiz.SetMagic(rookMagic[:], SquareF4, 9269586743957521409, 10)
	wiz.SetMagic(rookMagic[:], SquareF5, 140754676613633, 10)
	wiz.SetMagic(rookMagic[:], SquareF6, 8859435012, 10)
	wiz.SetMagic(rookMagic[:], SquareF7, 105622918137857, 10)
	wiz.SetMagic(rookMagic[:], SquareF8, 93452063091195905, 11)
	wiz.SetMagic(rookMagic[:], SquareG1, 3848292811265, 11)
	wiz.SetMagic(rookMagic[:], SquareG2, 9441796687501985793, 10)
	wiz.SetMagic(rookMagic[:], SquareG3, 668793341028205569, 10)
	wiz.SetMagic(rookMagic[:], SquareG4, 3503805114303512577, 10)
	wiz.SetMagic(rookMagic[:], SquareG5, 1441856117960359937, 10)
	wiz.SetMagic(rookMagic[:], SquareG6, 648529410319974401, 10)
	wiz.SetMagic(rookMagic[:], SquareG7, 13979322776982393857, 10)
	wiz.SetMagic(rookMagic[:], SquareG8, 13835060872780858369, 11)
	wiz.SetMagic(rookMagic[:], SquareH1, 4539788820801, 12)
	wiz.SetMagic(rookMagic[:], SquareH2, 2359886214407946241, 11)
	wiz.SetMagic(rookMagic[:], SquareH3, 27041389040664577, 11)
	wiz.SetMagic(rookMagic[:], SquareH4, 159429253169153, 11)
	wiz.SetMagic(rookMagic[:], SquareH5, 4613955963706147841, 11)
	wiz.SetMagic(rookMagic[:], SquareH6, 4611686019534716929, 11)
	wiz.SetMagic(rookMagic[:], SquareH7, 27025995845339137, 11)
	wiz.SetMagic(rookMagic[:], SquareH8, 633464726504577, 12)

	// enable the next line to find new magics
	// wiz.SearchMagics(rookMagic[:])
}

///////////////////////////////////////////////
// initBbSuperAttack : init super attacks

func initBbSuperAttack() {
	for sq := SquareMinValue; sq <= SquareMaxValue; sq++ {
		bbSuperAttack[sq] = slidingAttack(sq, rookDeltas, BbEmpty) | slidingAttack(sq, bishopDeltas, BbEmpty)
	}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// pieceToSymbolStr : converts piece to a string containing its symbol
// -> piece Piece : piece
// <- string : symbol as string ( unicode symbol if USE_UNICODE_SYMBOLS global setting is true )

func PieceToSymbolStr(piece Piece) string {
	if USE_UNICODE_SYMBOLS {
		return fmt.Sprintf("%c",pieceToSymbolU[piece])
	}
	return fmt.Sprintf("%c",pieceToSymbol[piece])
}
///////////////////////////////////////////////////

///////////////////////////////////////////////
// CastlingAbility : returns kings' castling ability

func (pos *Position) CastlingAbility() Castle {
	return pos.curr.CastlingAbility
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// NewPosition : returns a new position
// <- *Position : position

func NewPosition() *Position {
	pos := &Position{
		fullmoveCounter: 1,
		states:          make([]state, 1, 4),
	}
	pos.curr = &pos.states[pos.Ply]
	return pos
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// Put : puts a piece on the board
// does nothing if pi is NoPiece, does not validate input
// -> pos *Position : position to set
// -> sq Square : square
// -> pi Piece : piece

func (pos *Position) Put(sq Square, pi Piece) {
	if pi != NoPiece {
		pos.curr.Zobrist ^= zobristPiece[pi][sq]
		bb := sq.Bitboard()
		pos.ByColor[pi.Color()] |= bb
		pos.ByFigure[pi.Figure()] |= bb
	}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// ByPiece : shortcut for ByColor[col]&ByFigure[fig]
// -> pos *Position : position
// -> col Color : color
// -> fig Figure : figure
// -> Bitboard : resulting bitboard

func (pos *Position) ByPiece(col Color, fig Figure) Bitboard {
	return pos.ByColor[col] & pos.ByFigure[fig]
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// SetSideToMove : sets the side to move, correctly updating the Zobrist key
// -> pos *Position : position to set
// -> col Color : color

func (pos *Position) SetSideToMove(col Color) {
	pos.curr.Zobrist ^= zobristColor[pos.SideToMove]
	pos.SideToMove = col
	pos.curr.Zobrist ^= zobristColor[pos.SideToMove]
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// SetCastlingAbility : sets the side to move, correctly updating the Zobrist key
// -> pos *Position : position to set
// -> castle Castle : castling rights

func (pos *Position) SetCastlingAbility(castle Castle) {
	if pos.curr.CastlingAbility == castle {
		return
	}

	pos.curr.Zobrist ^= zobristCastle[pos.curr.CastlingAbility]
	pos.curr.CastlingAbility = castle
	pos.curr.Zobrist ^= zobristCastle[pos.curr.CastlingAbility]
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// SetEnpassantSquare : sets the en passant square correctly updating the Zobrist key
// -> pos *Position : position to set
// -> sq Square : en passant square

func (pos *Position) SetEnpassantSquare(sq Square) {
	if sq == pos.curr.EnpassantSquare[1] {
		// in the trivial case both values are SquareA1
		// and zobrist value doesn't change
		return
	}

	pos.curr.Zobrist ^= zobristEnpassant[pos.curr.EnpassantSquare[0]]
	pos.curr.EnpassantSquare[0] = sq
	pos.curr.EnpassantSquare[1] = sq

	if sq != SquareA1 {
		// in polyglot the hash key for en passant is updated only if
		// an en passant capture is possible next move, in other words
		// if there is an enemy pawn next to the end square of the move
		var theirs Bitboard
		if sq.Rank() == 2 { // White
			theirs, sq = pos.ByPiece(Black, Pawn), RankFile(3, sq.File())
		} else if sq.Rank() == 5 { // Black
			theirs, sq = pos.ByPiece(White, Pawn), RankFile(4, sq.File())
		} else {
			if IS_Horde {
				// in horde there can be ep squares on rank 1 and 6
				if sq.Rank() == 1 { // White
					theirs, sq = pos.ByPiece(Black, Pawn), RankFile(2, sq.File())
				} else if sq.Rank() == 6 { // Black
					theirs, sq = pos.ByPiece(White, Pawn), RankFile(5, sq.File())
				} else {
					panic("bad horde en passant square")
				}
			} else {
				panic("bad en passant square")
			}
		}

		if (sq.File() == 0 || !theirs.Has(sq-1)) && (sq.File() == 7 || !theirs.Has(sq+1)) {
			pos.curr.EnpassantSquare[0] = SquareA1
		}
	}

	pos.curr.Zobrist ^= zobristEnpassant[pos.curr.EnpassantSquare[0]]
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// Get : returns the piece at square sq
// -> pos *Position : position
// -> sq Square : square
// <- Piece : piece

func (pos *Position) Get(sq Square) Piece {
	var col Color
	if pos.ByColor[White].Has(sq) {
		col = White
	} else if pos.ByColor[Black].Has(sq) {
		col = Black
	} else {
		return NoPiece
	}

	for fig := FigureMinValue; fig <= FigureMaxValue; fig++ {
		if pos.ByFigure[fig].Has(sq) {
			return ColorFigure(col, fig)
		}
	}
	if IS_Atomic {
		// TODO : in atomic sometimes square has color but no figure, has to be investigated
		// NoPiece is returned to avoid panic
		return NoPiece
	}
	panic("unreachable: square has color, but no figure")
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// ParsePiecePlacement : parse pieces from str (FEN like) into pos
// -> str string : pos string
// -> pos *Position : position to set
// <- error : error

func ParsePiecePlacement(str string, pos *Position) error {
	r, f := 0, 0
	for _, p := range str {
		if p == '/' {
			if r == 7 {
				return fmt.Errorf("expected 8 ranks")
			}
			if f != 8 {
				return fmt.Errorf("expected 8 squares per rank, got %d", f)
			}
			r, f = r+1, 0
			continue
		}

		if '1' <= p && p <= '8' {
			f += int(p) - int('0')
			continue
		}
		pi := symbolToPiece[p]
		if pi == NoPiece {
			return fmt.Errorf("expected rank or number, got %s", string(p))
		}
		if f >= 8 {
			return fmt.Errorf("rank %d too long (%d cells)", 8-r, f)
		}

		// 7-r because FEN describes the table from 8th rank.
		pos.Put(RankFile(7-r, f), pi)
		f++

	}

	if f < 8 {
		return fmt.Errorf("rank %d too short (%d cells)", r+1, f)
	}
	return nil
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// ParseSideToMove : sets side to move for pos from str
// -> str : pos string
// -> pos *Position : position to set
// <- error : error

func ParseSideToMove(str string, pos *Position) error {
	if col, ok := symbolToColor[str]; ok {
		pos.SetSideToMove(col)
		return nil
	}
	return fmt.Errorf("invalid color %s", str)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// ParseCastlingAbility : sets castling ability for pos from str
// -> str string : position string
// -> pos *Position : position to set
// <- error : error

func ParseCastlingAbility(str string, pos *Position) error {
	if str == "-" {
		pos.SetCastlingAbility(NoCastle)
		return nil
	}

	ability := NoCastle
	for _, p := range str {
		info, ok := symbolToCastleInfo[p]
		if !ok {
			return fmt.Errorf("invalid castling ability %s", str)
		}
		ability |= info.Castle
		for i := 0; i < 2; i++ {
			if info.Piece[i] != pos.Get(info.Square[i]) {
				return fmt.Errorf("expected %v at %v, got %v",
					info.Piece[i], info.Square[i], pos.Get(info.Square[i]))
			}
		}
	}
	pos.SetCastlingAbility(ability)
	return nil
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// ParseEnpassantSquare : parses the en passant square from str
// -> str string : position string
// -> pos *Position : position to set
// <- error : error

func ParseEnpassantSquare(str string, pos *Position) error {
	if str[:1] == "-" {
		pos.SetEnpassantSquare(SquareA1)
		return nil
	}
	sq, err := SquareFromString(str)
	if err != nil {
		return err
	}
	pos.SetEnpassantSquare(sq)
	return nil
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// PositionFromFEN : parses fen and returns the position
// fen must contain the position using Forsyth–Edwards Notation
// http://en.wikipedia.org/wiki/Forsyth%E2%80%93Edwards_Notation
//
// rejects FEN with only four fields
// i.e. no full move counter or have move numberc
// -> fen string : fen
// <- *Position : position
// <- error : error

func PositionFromFEN(fen string) (*Position, error) {
	// pplit fen into 6 fields
	// same as string.Fields() but creates much less garbage
	// the optimization is important when a huge number of positions
	// need to be evaluated
	f, p := [6]string{}, 0
	for i := 0; i < len(fen); {
		// find the start and end of the token
		for ; i < len(fen) && fen[i] == ' '; i++ {
		}
		start := i
		for ; i < len(fen) && fen[i] != ' '; i++ {
		}
		limit := i

		if start == limit {
			continue
		}
		if p >= len(f) {
			return nil, fmt.Errorf("fen has too many fields")
		}
		f[p] = fen[start:limit]
		p++
	}
	if p < len(f) {
		return nil, fmt.Errorf("fen has too few fields")
	}

	// parse each field
	pos := NewPosition()
	if err := ParsePiecePlacement(f[0], pos); err != nil {
		return nil, err
	}
	if err := ParseSideToMove(f[1], pos); err != nil {
		return nil, err
	}
	if err := ParseCastlingAbility(f[2], pos); err != nil {
		return nil, err
	}
	if err := ParseEnpassantSquare(f[3], pos); err != nil {
		return nil, err
	}
	var err error
	if pos.curr.HalfmoveClock, err = strconv.Atoi(f[4]); err != nil {
		return nil, err
	}
	if pos.fullmoveCounter, err = strconv.Atoi(f[5]); err != nil {
		return nil, err
	}
	pos.Ply = (pos.fullmoveCounter - 1) * 2
	if pos.SideToMove == Black {
		pos.Ply++
	}
	return pos, nil
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// FormatPiecePlacement : converts a position to FEN piece placement
// -> pos *Position : position
// <- string : position string

func FormatPiecePlacement(pos *Position) string {
	s := ""
	for r := 7; r >= 0; r-- {
		space := 0
		for f := 0; f < 8; f++ {
			sq := RankFile(r, f)
			pi := pos.Get(sq)
			if pi == NoPiece {
				space++
			} else {
				if space != 0 {
					s += itoa[space:][:1]
					space = 0
				}
				s += pieceToSymbol[pi:][:1]
			}
		}

		if space != 0 {
			s += itoa[space:][:1]
		}
		if r != 0 {
			s += "/"
		}
	}
	return s
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// FormatEnpassantSquare : converts position's en passant square to string
// -> pos *Position : position
// <- string : en passant square as string

func FormatEnpassantSquare(pos *Position) string {
	if pos.EnpassantSquare() != SquareA1 {
		return pos.EnpassantSquare().String()
	}
	return "-"
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// FormatSideToMove : returns "w" for white to play or "b" for black to play
// -> pos *Position : position
// <- string : side letter

func FormatSideToMove(pos *Position) string {
	return colorToSymbol[pos.SideToMove:][:1]
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// FormatCastlingAbility : returns a string specifying the castling ability
// -> pos *Position : position
// <- string : castling rights as string, using standard FEN format

func FormatCastlingAbility(pos *Position) string {
	return pos.CastlingAbility().String()
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// IsOnBaseRank : is side's king on base rank
// -> color Color : side
// <- bool : true if on base rank

func (pos *Position) IsOnBaseRank(color Color) bool {
	kingPos := pos.ByFigure[King] & pos.ByColor[color]
	if (BbRank8 & kingPos) != 0 {
		return true
	}
	return false
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// PrintBoard : prints board for position
// -> pos *Position : position

func (pos *Position) PrintBoard() {
	/*fmt.Printf("  0123456701234567012345670123456701234567012345670123456701234567\n")
	for j:=0 ; j<FigureArraySize; j++ {
		fmt.Printf("%d %064b\n",j,pos.ByFigure[j])
	}*/
	var mask uint64=1 << uint(SquareArraySize-1)
	buff:=""
	for i:=0; i<SquareArraySize; i++ {
		found:=false
		var j=0
		for ; j<FigureArraySize; j++ {
			if uint64(pos.ByFigure[j]) & mask != 0 {
				found=true
				break
			}
		}
		if(!found){
			buff=PieceToSymbolStr(NoPiece)+buff
		} else {
			if uint64(pos.ByColor[Black]) & mask != 0 {
				buff=PieceToSymbolStr(Piece(2*j))+buff
			} else {
				buff=PieceToSymbolStr(Piece(2*j+1))+buff
			}
		}
		if (i%8) == 7 {
			fmt.Printf("%s\n",buff)
			buff=""
		}
		mask=mask >> 1
	}
	/*fmt.Printf("evalw %d evalb %d eval %d bw %v bb %v\n",
		EvaluateSideRk(pos, White),EvaluateSideRk(pos, Black),Evaluate(pos),
		pos.IsOnBaseRank(White),pos.IsOnBaseRank(Black))*/
	/*pos.GetKingBitboard(White).Print()
	pos.GetKingBitboard(Black).Print()*/
	//fmt.Printf("kings adjacent %v\n",pos.KingsAdjacent())
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// Print : prints bitboard
// -> bb Bitboard : position

func (bb Bitboard) Print() {
	var mask uint64=1 << uint(SquareArraySize-1)
	buff:=""
	for i:=0; i<SquareArraySize; i++ {
		found := (uint64(bb) & mask) != 0
		if(!found){
			buff="0"+buff
		} else {
			buff="1"+buff
		}
		if (i%8) == 7 {
			fmt.Printf("%s\n",buff)
			buff=""
		}
		mask=mask >> 1
	}
	fmt.Println()
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// String : returns position in FEN format
// for table format use PrettyPrint
// -> pos *Position : position
// <- string : FEN string

func (pos *Position) String() string {
	s := FormatPiecePlacement(pos)
	s += " " + FormatSideToMove(pos)
	s += " " + FormatCastlingAbility(pos)
	s += " " + FormatEnpassantSquare(pos)
	s += " " + strconv.Itoa(pos.curr.HalfmoveClock)
	s += " " + strconv.Itoa(pos.fullmoveCounter)
	return s
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// prev : returns state at previous ply
// -> pos *Position : position
// <- *state : state

func (pos *Position) prev() *state {
	return &pos.states[len(pos.states)-1]
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// GetNoStates : number of states in position
// <- int : number of states

func (pos *Position) GetNoStates() int {
	return len(pos.states)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// popState : pops one ply
// -> pos *Position : position

func (pos *Position) popState() {
	len := len(pos.states) - 1
	pos.states = pos.states[:len]
	pos.curr = &pos.states[len-1]
	pos.Ply--
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// pushState : adds one ply
// -> pos *Position : position

func (pos *Position) pushState() {
	len := len(pos.states)
	pos.states = append(pos.states, pos.states[len-1])
	pos.curr = &pos.states[len]
	pos.Ply++
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// FullmoveCounter : returns fullmovecounter
// -> pos *Position : position
// <- int : full move counter

func (pos *Position) FullmoveCounter() int {
	return pos.fullmoveCounter
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// SetFullmoveCounter : sets full move counter
// -> pos *Position : position
// -> n int : counter

func (pos *Position) SetFullmoveCounter(n int) {
	pos.fullmoveCounter = n
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// HalfmoveClock : returns current half move clock
// -> pos *Position : position
// <- int : half move clock

func (pos *Position) HalfmoveClock() int {
	return pos.curr.HalfmoveClock
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// SetHalfmoveClock : sets half move clock
// -> pos *Position : position
// -> n int : half move clock 

func (pos *Position) SetHalfmoveClock(n int) {
	pos.curr.HalfmoveClock = n
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// IsEnpassantSquare : returns true if sq is the en passant square
// -> pos *Position : position
// -> sq Square : square to check
// <- bool : true if square is en passant square

func (pos *Position) IsEnpassantSquare(sq Square) bool {
	return sq != SquareA1 && sq == pos.EnpassantSquare()
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// EnpassantSquare : returns the en passant square
// -> pos *Position : position
// <- Square : en passant square

func (pos *Position) EnpassantSquare() Square {
	return pos.curr.EnpassantSquare[1]
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// LastMove : returns the last move played, if any
// -> pos *Position : position
// <- Move : last move

func (pos *Position) LastMove() Move {
	return pos.curr.Move
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// Zobrist : returns the zobrist key of the position
// the returned value is equal to polyglot book key
// (http://hgm.nubati.net/book_format.html)
// -> pos *Position : position
// <- uint64 : Zobrist key

func (pos *Position) Zobrist() uint64 {
	return pos.curr.Zobrist
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// NumNonPawns : returns the number of minor and major pieces
// -> pos *Position : position
// -> col Color : side
// <- int : number of pieces

func (pos *Position) NumNonPawns(col Color) int {
	return int((pos.ByColor[col] &^ pos.ByFigure[Pawn] &^ pos.ByFigure[King]).Count())
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// HasNonPawns : returns whether col has at least some minor or major pieces
// -> pos *Position : position
// -> col Color : side
// <- bool : true if side has some pieces

func (pos *Position) HasNonPawns(col Color) bool {
	return pos.ByColor[col]&^pos.ByFigure[Pawn]&^pos.ByFigure[King] != 0
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// GetAttacker : returns the smallest figure of color them that attacks sq
// -> pos *Position : position
// -> sq Square : square
// -> them Color : them color
// <- Figure : smallest attacking figure

func (pos *Position) GetAttacker(sq Square, them Color) Figure {
	enemy := pos.ByColor[them]
	// Pawn
	if enemy&bbPawnAttack[sq]&pos.ByFigure[Pawn] != 0 {
		if att := sq.Bitboard() & pos.PawnThreats(them); att != 0 {
			return Pawn
		}
	}
	// Knight
	if enemy&bbKnightAttack[sq]&pos.ByFigure[Knight] != 0 {
		return Knight
	}
	// Quick test of queen's attack on an empty board.
	// Exclude pawns and knights because they were already tested.
	enemy &^= pos.ByFigure[Pawn]
	enemy &^= pos.ByFigure[Knight]
	if enemy&bbSuperAttack[sq] == 0 {
		return NoFigure
	}
	// Bishop
	all := pos.ByColor[White] | pos.ByColor[Black]
	bishop := BishopMobility(sq, all)
	if enemy&pos.ByFigure[Bishop]&bishop != 0 {
		return Bishop
	}
	// Rook
	rook := RookMobility(sq, all)
	if enemy&pos.ByFigure[Rook]&rook != 0 {
		return Rook
	}
	// Queen
	if enemy&pos.ByFigure[Queen]&(bishop|rook) != 0 {
		return Queen
	}
	// King.
	if enemy&bbKingAttack[sq]&pos.ByFigure[King] != 0 {
		return King
	}
	return NoFigure
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// PawnThreats : returns the set of squares threatened by side's pawns
// -> pos *Position : position
// -> side Color : side
// <- Bitboard : squares attacked by side's pawns

func (pos *Position) PawnThreats(side Color) Bitboard {
	pawns := Forward(side, pos.ByPiece(side, Pawn))
	return West(pawns) | East(pawns)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// IsPseudoLegal : returns true if m is a pseudo legal move for pos
// it returns true iff m can be executed even if own king is in check
// after the move, NullMove is not a valid move
// -> pos *Position : position
// -> m Move : move
// <- bool : true if pseudo legal

func (pos *Position) IsPseudoLegal(m Move) bool {
	if m == NullMove ||
		m.SideToMove() != pos.SideToMove ||
		pos.Get(m.From()) != m.Piece() ||
		pos.Get(m.CaptureSquare()) != m.Capture() ||
		m.Piece().Color() == m.Capture().Color() {
		return false
	}

	if m.Piece().Figure() == Pawn {
		// Pawn move is tested above. Promotion is always correct.
		if m.MoveType() == Enpassant && !pos.IsEnpassantSquare(m.To()) {
			return false
		}
		if BbPawnStartRank.Has(m.From()) && BbPawnDoubleRank.Has(m.To()) && !pos.IsEmpty((m.From()+m.To())/2) {
			return false
		}
		return true
	}
	if m.Piece().Figure() == Knight {
		// Knight move is tested above. Knight jumps around.
		return true
	}

	// Quick test of queen's attack on an empty board.
	sq := m.From()
	to := m.To().Bitboard()
	if bbSuperAttack[sq]&to == 0 {
		return false
	}

	all := pos.ByColor[White] | pos.ByColor[Black]

	switch m.Piece().Figure() {
	case Pawn: // handled aove
		panic("unreachable")
	case Knight: // handled above
		panic("unreachable")
	case Bishop:
		return to&BishopMobility(sq, all) != 0
	case Rook:
		return to&RookMobility(sq, all) != 0
	case Queen:
		return to&QueenMobility(sq, all) != 0
	case King:
		if m.MoveType() == Normal {
			return to&bbKingAttack[sq] != 0
		}

		// m.MoveType() == Castling
		if m.SideToMove() == White && m.To() == SquareG1 {
			if pos.CastlingAbility()&WhiteOO == 0 ||
				!pos.IsEmpty(SquareF1) || !pos.IsEmpty(SquareG1) {
				return false
			}
		}
		if m.SideToMove() == White && m.To() == SquareC1 {
			if pos.CastlingAbility()&WhiteOOO == 0 ||
				!pos.IsEmpty(SquareB1) ||
				!pos.IsEmpty(SquareC1) ||
				!pos.IsEmpty(SquareD1) {
				return false
			}
		}
		if m.SideToMove() == Black && m.To() == SquareG8 {
			if pos.CastlingAbility()&BlackOO == 0 ||
				!pos.IsEmpty(SquareF8) ||
				!pos.IsEmpty(SquareG8) {
				return false
			}
		}
		if m.SideToMove() == Black && m.To() == SquareC8 {
			if pos.CastlingAbility()&BlackOOO == 0 ||
				!pos.IsEmpty(SquareB8) ||
				!pos.IsEmpty(SquareC8) ||
				!pos.IsEmpty(SquareD8) {
				return false
			}
		}
		rook, start, end := CastlingRook(m.To())
		if pos.Get(start) != rook {
			return false
		}
		them := m.SideToMove().Opposite()
		if pos.GetAttacker(m.From(), them) != NoFigure ||
			pos.GetAttacker(end, them) != NoFigure ||
			pos.GetAttacker(m.To(), them) != NoFigure {
			return false
		}
	default:
		panic("unreachable")
	}

	return true
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// IsEmpty : returns true if there is no piece at sq
// -> pos *Position : position
// -> sq Square : square
// <- bool : true if empty

func (pos *Position) IsEmpty(sq Square) bool {
	return !(pos.ByColor[White] | pos.ByColor[Black]).Has(sq)
}

///////////////////////////////////////////////


///////////////////////////////////////////////////
// OLD

///////////////////////////////////////////////////
// HasLegalMoves : returns true if current side has any legal moves
// this function is very expensive
// -> pos *Position : position
// <- bool : true if has legal moves

/*func (pos *Position) HasLegalMoves() bool {
	var moves []Move
	pos.GenerateMoves(All, &moves)
	us := pos.SideToMove

	for _, m := range moves {
		pos.DoMove(m)
		checked := pos.IsChecked(us)
		pos.UndoMove()

		if !checked {
			return true
		}
	}

	return false
}*/

///////////////////////////////////////////////////

// END OLD
///////////////////////////////////////////////////

///////////////////////////////////////////////////
// NEW

///////////////////////////////////////////////////
// HasLegalMoves : returns true if current side has any legal moves
// this function is very expensive
// now we use GetLegalMoves for this
// -> pos *Position : position
// <- bool : true if has legal moves

func (pos *Position) HasLegalMoves() bool {
	numMoves := len(pos.GetLegalMoves(GET_FIRST))
	return numMoves > 0
}

///////////////////////////////////////////////////

///////////////////////////////////////////////////
// GetLegalMoves : generate all legal moves
// -> pos *Position : position
// -> getfirst bool : only get the first legal move
// <- []Move : legal move list

func (pos *Position) GetLegalMoves(getfirst bool) []Move {
	var moves []Move
	var legalMoves=[]Move{}
	pos.GenerateMoves(All, &moves)
	us := pos.SideToMove
	them := us.Opposite()

	for _, m := range moves {
		pos.DoMove(m)
		checked := pos.IsChecked(us)
		// In Racing Kings any move that gives local check is also illegal.
		if IS_Racing_Kings {
			checkedThem := pos.IsCheckedLocal(them)
			checked=checked||checkedThem
		}
		pos.UndoMove()

		if !checked {
			if getfirst {	
				return []Move{m}
			} else {
				legalMoves=append(legalMoves,m)
			}
		}
	}

	return legalMoves
}

///////////////////////////////////////////////////

///////////////////////////////////////////////////
// PrintLegalMoves : print legal moves
// -> pos *Position

func (pos *Position) PrintLegalMoves() {
	moves := pos.GetLegalMoves(GET_ALL)
	for i, move := range moves {
		fmt.Printf("%2d %6s ",i+1,move.LAN())
		if ((i%8)==7) && (i!=(len(moves)-1)) {
			fmt.Printf("\n")
		}
	}
	fmt.Printf("\n")
}

///////////////////////////////////////////////////

// END NEW
///////////////////////////////////////////////////

///////////////////////////////////////////////////
// UndoMove : takes back the last move
// -> pos *Position : position

func (pos *Position) UndoMove() {
	move := pos.LastMove()
	pos.SetSideToMove(pos.SideToMove.Opposite())
	// CastlingAbility and EnpassantSquare are restored by pos.popState()
	pos.SetCastlingAbility(pos.prev().CastlingAbility)
	pos.SetEnpassantSquare(pos.prev().EnpassantSquare[1])

	// modify the chess board
	pi := move.Piece()
	pos.Put(move.From(), pi)
	if IS_Atomic && ( pos.curr.NumExplosions > 0 ) {
		// if there was an atomic capture the piece was already removed by explosion
	} else {
		pos.Remove(move.To(), move.Target())
	}
	pos.Put(move.CaptureSquare(), move.Capture())

	if IS_Atomic && ( pos.curr.NumExplosions > 0 ) {
		for i := 0 ; i < pos.curr.NumExplosions ; i++ {
			esq := pos.curr.ExplosionInfo[i].sq
			epi := pos.curr.ExplosionInfo[i].piece
			pos.Put(esq, epi)
		}
	}

	// move rook on castling
	if move.MoveType() == Castling {
		rook, start, end := CastlingRook(move.To())
		pos.Put(start, rook)
		pos.Remove(end, rook)
	}

	if pos.SideToMove == Black {
		pos.fullmoveCounter--
	}

	pos.popState()
}

///////////////////////////////////////////////////

///////////////////////////////////////////////////
// genKingMovesNear : generate king move near
// -> pos *Position : position
// -> mask Bitboard : mask
// -> moves *[]Move : moves

func (pos *Position) genKingMovesNear(mask Bitboard, moves *[]Move) {
	if IS_Horde {
		if pos.SideToMove == HORDE_Pawns_Side {
			// no king moves for the pawns in horde
			return
		}
	}
	pi := ColorFigure(pos.SideToMove, King)
	from := pos.ByPiece(pos.SideToMove, King).AsSquare()
	att := bbKingAttack[from] & mask
	pos.genBitboardMoves(pi, from, att, moves)
}

///////////////////////////////////////////////////

///////////////////////////////////////////////////
// genBitboardMoves : generate bitboard moves
// -> pos *Position : position
// -> pi Piece : piece
// -> from Square : from square
// -> att Bitboard : attack bitboard
// -> moves *[]Move : moves

func (pos *Position) genBitboardMoves(pi Piece, from Square, att Bitboard, moves *[]Move) {
	for att != 0 {
		to := att.Pop()
		*moves = append(*moves, MakeMove(Normal, from, to, pos.Get(to), pi))
	}
}

///////////////////////////////////////////////////

///////////////////////////////////////////////////
// genPawnDoubleAdvanceMoves : moves pawns two square
// -> pos *Position : position
// -> kind int : kind
// -> moves *[]Move : moves

func (pos *Position) genPawnDoubleAdvanceMoves(kind int, moves *[]Move) {
	if kind&Quiet == 0 {
		return
	}

	ours := pos.ByPiece(pos.SideToMove, Pawn)
	occu := pos.ByColor[White] | pos.ByColor[Black]
	pawn := ColorFigure(pos.SideToMove, Pawn)

	var forward Square
	if pos.SideToMove == White {
		ours &= RankBb(1) &^ South(occu) &^ South(South(occu))
		forward = RankFile(+2, 0)
	} else {
		ours &= RankBb(6) &^ North(occu) &^ North(North(occu))
		forward = RankFile(-2, 0)
	}

	for ours != 0 {
		from := ours.Pop()
		to := from + forward
		*moves = append(*moves, MakeMove(Normal, from, to, NoPiece, pawn))
	}

	if IS_Horde {
		// in horde pawns can move two squares from base rank
		ours = pos.ByPiece(pos.SideToMove, Pawn)
		occu = pos.ByColor[White] | pos.ByColor[Black]
		pawn = ColorFigure(pos.SideToMove, Pawn)

		if pos.SideToMove == White {
			ours &= RankBb(0) &^ South(occu) &^ South(South(occu))
			forward = RankFile(+2, 0)
		} else {
			ours &= RankBb(7) &^ North(occu) &^ North(North(occu))
			forward = RankFile(-2, 0)
		}

		for ours != 0 {
			from := ours.Pop()
			to := from + forward
			*moves = append(*moves, MakeMove(Normal, from, to, NoPiece, pawn))
		}
	}
}

///////////////////////////////////////////////////

///////////////////////////////////////////////////
// genBishopMoves : generate bishop moves
// -> pos *Position : position
// -> fig Figure : figure
// -> mask Bitboard : mask
// -> moves *[]Move

func (pos *Position) genBishopMoves(fig Figure, mask Bitboard, moves *[]Move) {
	pi := ColorFigure(pos.SideToMove, fig)
	ref := pos.ByColor[White] | pos.ByColor[Black]
	for bb := pos.ByPiece(pos.SideToMove, fig); bb != 0; {
		from := bb.Pop()
		att := bishopMagic[from].Attack(ref) & mask
		pos.genBitboardMoves(pi, from, att, moves)
	}
}

///////////////////////////////////////////////////

///////////////////////////////////////////////////
// genRookMoves : generate rook moves
// -> pos *Position : position
// -> fig Figure : figure
// -> mask Bitboard : mask
// -> moves *[]Move

func (pos *Position) genRookMoves(fig Figure, mask Bitboard, moves *[]Move) {
	pi := ColorFigure(pos.SideToMove, fig)
	ref := pos.ByColor[White] | pos.ByColor[Black]
	for bb := pos.ByPiece(pos.SideToMove, fig); bb != 0; {
		from := bb.Pop()
		att := rookMagic[from].Attack(ref) & mask
		pos.genBitboardMoves(pi, from, att, moves)
	}
}

///////////////////////////////////////////////////

///////////////////////////////////////////////////
// -> pos *Position : position
// -> to Square : to square
// <- MoveType : move type
// <- Piece : piece

func (pos *Position) pawnCapture(to Square) (MoveType, Piece) {
	if pos.IsEnpassantSquare(to) {
		return Enpassant, ColorFigure(pos.SideToMove.Opposite(), Pawn)
	}
	return Normal, pos.Get(to)
}

///////////////////////////////////////////////////

///////////////////////////////////////////////////
// genPawnAdvanceMoves : moves pawns one square
// does not generate promotions
// -> pos *Position : position
// kind int : kind
// moves *[]Move : moves

func (pos *Position) genPawnAdvanceMoves(kind int, moves *[]Move) {
	if kind&Quiet == 0 {
		return
	}

	ours := pos.ByPiece(pos.SideToMove, Pawn)
	occu := pos.ByColor[White] | pos.ByColor[Black]
	pawn := ColorFigure(pos.SideToMove, Pawn)

	var forward Square
	if pos.SideToMove == White {
		ours = ours &^ South(occu) &^ BbRank7
		forward = RankFile(+1, 0)
	} else {
		ours = ours &^ North(occu) &^ BbRank2
		forward = RankFile(-1, 0)
	}

	for ours != 0 {
		from := ours.Pop()
		to := from + forward
		*moves = append(*moves, MakeMove(Normal, from, to, NoPiece, pawn))
	}
}

///////////////////////////////////////////////////

///////////////////////////////////////////////////
// genPawnPromotions : generate pawn promotions
// -> pos *Position : position
// kind int : kind
// moves *[]Move : moves

func (pos *Position) genPawnPromotions(kind int, moves *[]Move) {
	if kind&(Violent|Tactical) == 0 {
		return
	}

	// minimum and maximum promotion pieces
	// Tactical -> Knight - Rook
	// Violent -> Queen
	pMin, pMax := Queen, Rook
	if kind&Violent != 0 {
		pMax = Queen
	}
	if kind&Tactical != 0 {
		pMin = Knight
	}

	us := pos.SideToMove
	them := us.Opposite()

	// get the pawns that can be promoted
	all := pos.ByColor[White] | pos.ByColor[Black]
	ours := pos.ByPiece(us, Pawn)
	theirs := pos.ByColor[them] // their pieces

	forward := Square(0)
	if us == White {
		ours &= BbRank7
		forward = RankFile(+1, 0)
	} else {
		ours &= BbRank2
		forward = RankFile(-1, 0)
	}

	for ours != 0 {
		from := ours.Pop()
		to := from + forward

		if !all.Has(to) { // advance front
			for p := pMin; p <= pMax; p++ {
				*moves = append(*moves, MakeMove(Promotion, from, to, NoPiece, ColorFigure(us, p)))
			}
		}
		if to.File() != 0 && theirs.Has(to-1) { // take west
			capt := pos.Get(to - 1)
			for p := pMin; p <= pMax; p++ {
				*moves = append(*moves, MakeMove(Promotion, from, to-1, capt, ColorFigure(us, p)))
			}
		}
		if to.File() != 7 && theirs.Has(to+1) { // take east
			capt := pos.Get(to + 1)
			for p := pMin; p <= pMax; p++ {
				*moves = append(*moves, MakeMove(Promotion, from, to+1, capt, ColorFigure(us, p)))
			}
		}
	}
}

///////////////////////////////////////////////////

///////////////////////////////////////////////////
// genPawnAttackMoves : generate pawn attacks moves
// does not generate promotions
// -> pos *Position : position
// -> kind int : kind
// -> moves *[]Move : moves

func (pos *Position) genPawnAttackMoves(kind int, moves *[]Move) {
	if kind&Violent == 0 {
		return
	}

	theirs := pos.ByColor[pos.SideToMove.Opposite()]
	if pos.curr.EnpassantSquare[0] != SquareA1 {
		theirs |= pos.curr.EnpassantSquare[0].Bitboard()
	}

	forward := 0
	pawn := ColorFigure(pos.SideToMove, Pawn)
	ours := pos.ByPiece(pos.SideToMove, Pawn)
	if pos.SideToMove == White {
		ours = ours &^ BbRank7
		theirs = South(theirs)
		forward = +1
	} else {
		ours = ours &^ BbRank2
		theirs = North(theirs)
		forward = -1
	}

	// Left
	att := RankFile(forward, -1)
	for bbl := ours & East(theirs); bbl > 0; {
		from := bbl.Pop()
		to := from + att
		mt, capt := pos.pawnCapture(to)
		*moves = append(*moves, MakeMove(mt, from, to, capt, pawn))
	}

	// Right
	att = RankFile(forward, +1)
	for bbr := ours & West(theirs); bbr > 0; {
		from := bbr.Pop()
		to := from + att
		mt, capt := pos.pawnCapture(to)
		*moves = append(*moves, MakeMove(mt, from, to, capt, pawn))
	}
}

///////////////////////////////////////////////////

///////////////////////////////////////////////////
// genKnightMoves : generate knight moves
// -> pos *Position : position
// -> moves *[]Move : moves

func (pos *Position) genKnightMoves(mask Bitboard, moves *[]Move) {
	pi := ColorFigure(pos.SideToMove, Knight)
	for bb := pos.ByPiece(pos.SideToMove, Knight); bb != 0; {
		from := bb.Pop()
		att := bbKnightAttack[from] & mask
		pos.genBitboardMoves(pi, from, att, moves)
	}
}

///////////////////////////////////////////////////
// genKingCastles : generate king castles
// -> pos *Position : position
// -> kind int : kind
// -> moves *[]Move : moves

func (pos *Position) genKingCastles(kind int, moves *[]Move) {
	if kind&Tactical == 0 {
		return
	}

	rank := pos.SideToMove.KingHomeRank()
	oo, ooo := WhiteOO, WhiteOOO
	if pos.SideToMove == Black {
		oo, ooo = BlackOO, BlackOOO
	}

	// Castle king side.
	if pos.curr.CastlingAbility&oo != 0 {
		r5 := RankFile(rank, 5)
		r6 := RankFile(rank, 6)
		if !pos.IsEmpty(r5) || !pos.IsEmpty(r6) {
			goto EndCastleOO
		}

		r4 := RankFile(rank, 4)
		other := pos.SideToMove.Opposite()
		if pos.GetAttacker(r4, other) != NoFigure ||
			pos.GetAttacker(r5, other) != NoFigure ||
			pos.GetAttacker(r6, other) != NoFigure {
			goto EndCastleOO
		}

		*moves = append(*moves, MakeMove(Castling, r4, r6, NoPiece, ColorFigure(pos.SideToMove, King)))
	}
EndCastleOO:

	// castle queen side
	if pos.curr.CastlingAbility&ooo != 0 {
		r3 := RankFile(rank, 3)
		r2 := RankFile(rank, 2)
		r1 := RankFile(rank, 1)
		if !pos.IsEmpty(r3) || !pos.IsEmpty(r2) || !pos.IsEmpty(r1) {
			goto EndCastleOOO
		}

		r4 := RankFile(rank, 4)
		other := pos.SideToMove.Opposite()
		if pos.GetAttacker(r4, other) != NoFigure ||
			pos.GetAttacker(r3, other) != NoFigure ||
			pos.GetAttacker(r2, other) != NoFigure {
			goto EndCastleOOO
		}

		*moves = append(*moves, MakeMove(Castling, r4, r2, NoPiece, ColorFigure(pos.SideToMove, King)))
	}
EndCastleOOO:
}

///////////////////////////////////////////////////

///////////////////////////////////////////////////
// getMask : get mask
// -> pos *Position : position
// <- Bitboard : mask

func (pos *Position) getMask(kind int) Bitboard {
	mask := Bitboard(0)
	if kind&Violent != 0 {
		// generate all attacks
		// promotions are handled specially
		mask |= pos.ByColor[pos.SideToMove.Opposite()]
	}
	if kind&Quiet != 0 {
		// generate all non-attacks
		mask |= ^(pos.ByColor[White] | pos.ByColor[Black])
	}
	// tactical is handled specially
	return mask
}

///////////////////////////////////////////////////

///////////////////////////////////////////////////
// Remove : removes a piece from the table
// does nothing if pi is NoPiece, does not validate input
// -> pos *Position : position
// -> sq Square : square
// -> pi Piece : piece

func (pos *Position) Remove(sq Square, pi Piece) {
	if pi != NoPiece {
		pos.curr.Zobrist ^= zobristPiece[pi][sq]
		bb := ^sq.Bitboard()
		pos.ByColor[pi.Color()] &= bb
		pos.ByFigure[pi.Figure()] &= bb
	}
}

///////////////////////////////////////////////////

///////////////////////////////////////////////////
// IsCheckedLocal : is the side in local check
// only makes a difference in Racing Kings, where reaching the base rank is global check
// -> pos *Position : position
// -> side Color : side
// <- bool : true if checked

func (pos *Position) IsCheckedLocal(side Color) bool {
	if IS_Horde {
		// in horde the pawns can be never in check
		if side == HORDE_Pawns_Side {
			return false
		}
	}
	if IS_Atomic {
		// no check with adjacent kings
		if pos.KingsAdjacent() {
			return false
		}
	}
	kingSq := pos.ByPiece(side, King).AsSquare()
	return pos.GetAttacker(kingSq, side.Opposite()) != NoFigure
}

///////////////////////////////////////////////////

///////////////////////////////////////////////////
// IsExploded : is the king of side exploded
// -> pos *Position : position
// -> side Color : side
// <- bool : true if exploded

func (pos *Position) IsExploded(side Color) bool {
	if pos.GetKingBitboard(side) == 0 {
		return true
	}
	return false
}

///////////////////////////////////////////////////

///////////////////////////////////////////////////
// AllPawnsCaptured : checks whether all pawns were captured for pawns side in horde
// -> pos *Position : position
// <- bool : true if all pawns were captured

func (pos *Position) AllPawnsCaptured() bool {
	return ( pos.ByFigure[Pawn] & pos.ByColor[HORDE_Pawns_Side] ) == 0
}

///////////////////////////////////////////////////


///////////////////////////////////////////////////
// IsChecked : returns true if side's king is checked
// -> pos *Position : position
// -> side Color : side
// <- bool : true if checked

func (pos *Position) IsChecked(side Color) bool {
	///////////////////////////////////////////////////
	// NEW
	// check Horde global checks
	if IS_Horde {
		// in horde losing all pawns for the pawns is global check
		if side == HORDE_Pawns_Side {
			if pos.AllPawnsCaptured() {
				return true
			}
		}
	}
	// check Atomic global checks
	if IS_Atomic {
		them := side.Opposite()
		if pos.IsExploded(side) {
			// if our king exploded, we are in check
			return true
		}
		if pos.IsExploded(them) {
			// if opponent's king exploded without our king exploding, we are not in check
			return false
		}
	}
	// check Racing Kings global checks
	if IS_Racing_Kings {
		onbb := pos.IsOnBaseRank(Black)
		onbw := pos.IsOnBaseRank(White)
		if onbb && onbw {
			// if both kings on base rank, there is no global check
		} else if (side==White) && onbb {
			// if black reached the base rank white is always in check
			return true
		} else if (side==Black) && onbw {
			// if white reached the base rank and black is not on base rank, black is in check
			if !pos.IsOnBaseRank(Black) {
				return true
			}
		}
		// if no automatic check is true, then return the normal check
	}
	// END NEW
	///////////////////////////////////////////////////
	return pos.IsCheckedLocal(side)
}

///////////////////////////////////////////////////

///////////////////////////////////////////////////
// DoMove : executes a legal move
// -> pos *Position : position
// -> move Move : move

func (pos *Position) DoMove(move Move) {
	pos.pushState()
	curr := pos.curr
	curr.Move = move

	// update castling rights
	pi := move.Piece()
	if pi != NoPiece { // nullmove cannot change castling ability
		pos.SetCastlingAbility(curr.CastlingAbility &^ lostCastleRights[move.From()] &^ lostCastleRights[move.To()])
	}
	// update fullmove counter
	if pos.SideToMove == Black {
		pos.fullmoveCounter++
	}
	// update halfmove clock
	curr.HalfmoveClock++
	if pi.Figure() == Pawn || move.Capture() != NoPiece {
		curr.HalfmoveClock = 0
	}
	// set Enpassant square for capturing
	if pi.Figure() == Pawn && move.From().Rank()^move.To().Rank() == 2 {
		pos.SetEnpassantSquare((move.From() + move.To()) / 2)
	} else if pos.EnpassantSquare() != SquareA1 {
		pos.SetEnpassantSquare(SquareA1)
	}
	// move rook on castling
	if move.MoveType() == Castling {
		rook, start, end := CastlingRook(move.To())
		pos.Remove(start, rook)
		pos.Put(end, rook)
	}

	// update the pieces on the chess board
	pos.Remove(move.From(), pi)
	pos.Remove(move.CaptureSquare(), move.Capture())
	pos.Put(move.To(), move.Target())

	curr.NumExplosions = 0

	if IS_Atomic && ( pi != NoPiece ) && ( move.Capture() != NoPiece ) {
		// capturing piece now explodes
		pos.Remove(move.To(), move.Target())
		tosq := move.To()
		toneighbours := explosionsquares[tosq]
		explcnt := 0
		for _, nsq := range toneighbours {
			// explosion may affect castling rights
			pos.SetCastlingAbility(curr.CastlingAbility &^ lostCastleRights[nsq])
			npi := pos.Get(nsq)
			if ( npi != NoPiece ) && ( npi.Figure() != Pawn ) {
				curr.ExplosionInfo[explcnt].sq = nsq
				curr.ExplosionInfo[explcnt].piece = npi
				pos.Remove(nsq, npi)
				explcnt++
			}
		}
		pos.curr.NumExplosions = explcnt
	}

	pos.SetSideToMove(pos.SideToMove.Opposite())
}

///////////////////////////////////////////////////

///////////////////////////////////////////////
// GenerateMoves : appends to moves all moves valid from pos
// the generated moves are pseudo-legal, i.e. they can leave the king in check
// kind is a combination of Quiet, Tactical or Violent.
// -> pos *Position : position
// -> kind int : kind
// -> moves *[]Move : move list

func (pos *Position) GenerateMoves(kind int, moves *[]Move) {
	mask := pos.getMask(kind)
	// Order of the moves is important because the last quiet
	// moves will be reduced less.  Current order was produced
	// by testing 20 random orders and picking the best.
	pos.genKingMovesNear(mask, moves)
	pos.genPawnDoubleAdvanceMoves(kind, moves)
	pos.genRookMoves(Rook, mask, moves)
	pos.genBishopMoves(Queen, mask, moves)
	pos.genPawnAttackMoves(kind, moves)
	pos.genPawnAdvanceMoves(kind, moves)
	pos.genPawnPromotions(kind, moves)
	pos.genKnightMoves(mask, moves)
	pos.genBishopMoves(Bishop, mask, moves)
	pos.genKingCastles(kind, moves)
	pos.genRookMoves(Queen, mask, moves)
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// GenerateFigureMoves : generate moves for a given figure
// the generated moves are pseudo-legal, i.e. they can leave the king in check
// kind is a combination of Quiet, Tactical or Violent
// -> pos *Position : position
// -> fig Figure : figure
// -> kind int : kind
// -> moves *[]Move : move list

func (pos *Position) GenerateFigureMoves(fig Figure, kind int, moves *[]Move) {
	mask := pos.getMask(kind)
	switch fig {
	case Pawn:
		pos.genPawnAdvanceMoves(kind, moves)
		pos.genPawnAttackMoves(kind, moves)
		pos.genPawnDoubleAdvanceMoves(kind, moves)
		pos.genPawnPromotions(kind, moves)
	case Knight:
		pos.genKnightMoves(mask, moves)
	case Bishop:
		pos.genBishopMoves(Bishop, mask, moves)
	case Rook:
		pos.genRookMoves(Rook, mask, moves)
	case Queen:
		pos.genBishopMoves(Queen, mask, moves)
		pos.genRookMoves(Queen, mask, moves)
	case King:
		pos.genKingMovesNear(mask, moves)
		pos.genKingCastles(kind, moves)
	}
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// SANToMove : converts a move from SAN format to internal representation
// SAN stands for standard algebraic notation and
// its description can be found in FIDE handbook
// The set of strings accepted is a slightly different.
//   x (capture) presence or correctness is ignored.
//   + (check) and # (checkmate) is ignored.
//   e.p. (enpassant) is ignored
// TODO: verify that the returned move is legal.
// -> pos *Position : position
// -> s string : san
// <- Move : move
// <- error : error

func (pos *Position) SANToMove(s string) (Move, error) {
	moveType := Normal
	rank, file := -1, -1 // from
	to := SquareA1
	capture := NoPiece
	target := NoPiece

	// s[b:e] is the part that still needs to be parsed
	b, e := 0, len(s)
	if b == e {
		return Move(0), errorWrongLength
	}
	// skip + (check) and # (checkmate) at the end
	for e > b && (s[e-1] == '#' || s[e-1] == '+') {
		e--
	}

	if s[b:e] == "o-o" || s[b:e] == "O-O" { // king side castling
		moveType = Castling
		if pos.SideToMove == White {
			rank, file = SquareE1.Rank(), SquareE1.File()
			to = SquareG1
			target = WhiteKing
		} else {
			rank, file = SquareE8.Rank(), SquareE8.File()
			to = SquareG8
			target = BlackKing
		}
	} else if s[b:e] == "o-o-o" || s[b:e] == "O-O-O" { // queen side castling
		moveType = Castling
		if pos.SideToMove == White {
			rank, file = SquareE1.Rank(), SquareE1.File()
			to = SquareC1
			target = WhiteKing
		} else {
			rank, file = SquareE8.Rank(), SquareE8.File()
			to = SquareC8
			target = BlackKing
		}
	} else { // all other moves
		// get the piece
		if ('a' <= s[b] && s[b] <= 'h') || s[b] == 'x' {
			target = ColorFigure(pos.SideToMove, Pawn)
		} else {
			if fig := symbolToFigure[rune(s[b])]; fig == NoFigure {
				return Move(0), errorUnknownFigure
			} else {
				target = ColorFigure(pos.SideToMove, fig)
			}
			b++
		}

		// skip e.p. when enpassant
		if e-4 > b && s[e-4:e] == "e.p." {
			e -= 4
		}

		// check pawn promotion
		if e-1 < b {
			return Move(0), errorWrongLength
		}
		if !('1' <= s[e-1] && s[e-1] <= '8') {
			// not a rank, but a promotion
			if target.Figure() != Pawn {
				return Move(0), errorBadPromotion
			}
			if fig := symbolToFigure[rune(s[e-1])]; fig == NoFigure {
				return Move(0), errorUnknownFigure
			} else {
				moveType = Promotion
				target = ColorFigure(pos.SideToMove, fig)
			}
			e--
			if e-1 >= b && s[e-1] == '=' {
				// sometimes = is inserted before promotion figure
				e--
			}
		}

		// handle destination square
		if e-2 < b {
			return Move(0), errorWrongLength
		}
		var err error
		to, err = SquareFromString(s[e-2 : e])
		if err != nil {
			return Move(0), err
		}
		if target.Figure() == Pawn && pos.IsEnpassantSquare(to) {
			moveType = Enpassant
			capture = ColorFigure(pos.SideToMove.Opposite(), Pawn)
		} else {
			capture = pos.Get(to)
		}
		e -= 2

		// ignore 'x' (capture) or '-' (no capture) if present
		if e-1 >= b && (s[e-1] == 'x' || s[e-1] == '-') {
			e--
		}

		// parse disambiguation
		if e-b > 2 {
			return Move(0), errorBadDisambiguation
		}
		for ; b < e; b++ {
			switch {
			case 'a' <= s[b] && s[b] <= 'h':
				file = int(s[b] - 'a')
			case '1' <= s[b] && s[b] <= '8':
				rank = int(s[b] - '1')
			default:
				return Move(0), errorBadDisambiguation
			}
		}
	}

	// loop through all moves and find out one that matches
	var moves []Move
	if moveType == Promotion {
		pos.GenerateFigureMoves(Pawn, All, &moves)
	} else {
		pos.GenerateFigureMoves(target.Figure(), All, &moves)
	}
	for _, pm := range moves {
		if pm.MoveType() != moveType || pm.Capture() != capture {
			continue
		}
		if pm.To() != to || pm.Target() != target {
			continue
		}
		if rank != -1 && pm.From().Rank() != rank {
			continue
		}
		if file != -1 && pm.From().File() != file {
			continue
		}
		return pm, nil
	}
	return Move(0), errorNoSuchMove
}

///////////////////////////////////////////////

///////////////////////////////////////////////
// UCIToMove : parses a move given in UCI format
// s can be "a2a4" or "h7h8Q" for pawn promotion
// -> pos *Position : position
// -> s string : uci move
// <- Move : move
// <- error : error

func (pos *Position) UCIToMove(s string) (Move, error) {
	if len(s) < 4 {
		return NullMove, fmt.Errorf("%s is too short", s)
	}

	from, err := SquareFromString(s[0:2])
	if err != nil {
		return NullMove, err
	}
	to, err := SquareFromString(s[2:4])
	if err != nil {
		return NullMove, err
	}

	moveType := Normal
	capt := pos.Get(to)
	target := pos.Get(from)

	pi := pos.Get(from)
	if pi.Figure() == Pawn && pos.IsEnpassantSquare(to) {
		moveType = Enpassant
		capt = ColorFigure(pos.SideToMove.Opposite(), Pawn)
	}
	if pi == WhiteKing && from == SquareE1 && (to == SquareC1 || to == SquareG1) {
		moveType = Castling
	}
	if pi == BlackKing && from == SquareE8 && (to == SquareC8 || to == SquareG8) {
		moveType = Castling
	}
	if pi.Figure() == Pawn && (to.Rank() == 0 || to.Rank() == 7) {
		if len(s) != 5 {
			return NullMove, fmt.Errorf("%s doesn't have a promotion piece", s)
		}
		moveType = Promotion
		target = ColorFigure(pos.SideToMove, symbolToFigure[rune(s[4])])
	} else {
		if len(s) != 4 {
			return NullMove, fmt.Errorf("%s move is too long", s)
		}
	}

	move := MakeMove(moveType, from, to, capt, target)
	if !pos.IsPseudoLegal(move) {
		return NullMove, fmt.Errorf("%s is not a valid move", s)
	}
	return move, nil
}

///////////////////////////////////////////////
