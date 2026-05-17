package dao

import (
	"fmt"

	"github.com/centrifuge/go-substrate-rpc-client/v4/scale"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/wetee-dao/ink.go/util"
)

type Tuple_4 struct { // Tuple
	F0 types.H160
	F1 types.U256
}
type Error struct { // Enum
	TokenNotFound         *bool // 0
	MemberExisted         *bool // 1
	MemberNotExisted      *bool // 2
	MemberBalanceNotZero  *bool // 3
	PublicJoinNotAllowed  *bool // 4
	LowBalance            *bool // 5
	InsufficientAllowance *bool // 6
	CallFailed            *bool // 7
	InvalidDeposit        *bool // 8
	TransferFailed        *bool // 9
	MustCallByGov         *bool // 10
	PropNotOngoing        *bool // 11
	PropNotEnd            *bool // 12
	InvalidProposal       *bool // 13
	InvalidProposalStatus *bool // 14
	InvalidProposalCaller *bool // 15
	InvalidDepositTime    *bool // 16
	InvalidVoteTime       *bool // 17
	InvalidVoteStatus     *bool // 18
	InvalidVoteUser       *bool // 19
	ProposalInDecision    *bool // 20
	VoteAlreadyUnlocked   *bool // 21
	InvalidVoteUnlockTime *bool // 22
	ProposalNotConfirmed  *bool // 23
	NoTrack               *bool // 24
	MaxBalanceOverflow    *bool // 25
	TransferDisable       *bool // 26
	InvalidVote           *bool // 27
	SetCodeFailed         *bool // 28
	SpendNotFound         *bool // 29
	SpendAlreadyExecuted  *bool // 30
	SpendTransferError    *bool // 31
	ReentrantCall         *bool // 32
}

func (ty Error) Encode(encoder scale.Encoder) (err error) {
	if ty.TokenNotFound != nil {
		err = encoder.PushByte(0)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.MemberExisted != nil {
		err = encoder.PushByte(1)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.MemberNotExisted != nil {
		err = encoder.PushByte(2)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.MemberBalanceNotZero != nil {
		err = encoder.PushByte(3)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.PublicJoinNotAllowed != nil {
		err = encoder.PushByte(4)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.LowBalance != nil {
		err = encoder.PushByte(5)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.InsufficientAllowance != nil {
		err = encoder.PushByte(6)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.CallFailed != nil {
		err = encoder.PushByte(7)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.InvalidDeposit != nil {
		err = encoder.PushByte(8)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.TransferFailed != nil {
		err = encoder.PushByte(9)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.MustCallByGov != nil {
		err = encoder.PushByte(10)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.PropNotOngoing != nil {
		err = encoder.PushByte(11)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.PropNotEnd != nil {
		err = encoder.PushByte(12)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.InvalidProposal != nil {
		err = encoder.PushByte(13)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.InvalidProposalStatus != nil {
		err = encoder.PushByte(14)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.InvalidProposalCaller != nil {
		err = encoder.PushByte(15)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.InvalidDepositTime != nil {
		err = encoder.PushByte(16)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.InvalidVoteTime != nil {
		err = encoder.PushByte(17)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.InvalidVoteStatus != nil {
		err = encoder.PushByte(18)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.InvalidVoteUser != nil {
		err = encoder.PushByte(19)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.ProposalInDecision != nil {
		err = encoder.PushByte(20)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.VoteAlreadyUnlocked != nil {
		err = encoder.PushByte(21)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.InvalidVoteUnlockTime != nil {
		err = encoder.PushByte(22)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.ProposalNotConfirmed != nil {
		err = encoder.PushByte(23)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.NoTrack != nil {
		err = encoder.PushByte(24)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.MaxBalanceOverflow != nil {
		err = encoder.PushByte(25)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.TransferDisable != nil {
		err = encoder.PushByte(26)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.InvalidVote != nil {
		err = encoder.PushByte(27)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.SetCodeFailed != nil {
		err = encoder.PushByte(28)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.SpendNotFound != nil {
		err = encoder.PushByte(29)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.SpendAlreadyExecuted != nil {
		err = encoder.PushByte(30)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.SpendTransferError != nil {
		err = encoder.PushByte(31)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.ReentrantCall != nil {
		err = encoder.PushByte(32)
		if err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("unrecognized enum")
}

func (ty *Error) Decode(decoder scale.Decoder) (err error) {
	variant, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}
	switch variant {
	case 0: // Base
		t := true
		ty.TokenNotFound = &t
		return
	case 1: // Base
		t := true
		ty.MemberExisted = &t
		return
	case 2: // Base
		t := true
		ty.MemberNotExisted = &t
		return
	case 3: // Base
		t := true
		ty.MemberBalanceNotZero = &t
		return
	case 4: // Base
		t := true
		ty.PublicJoinNotAllowed = &t
		return
	case 5: // Base
		t := true
		ty.LowBalance = &t
		return
	case 6: // Base
		t := true
		ty.InsufficientAllowance = &t
		return
	case 7: // Base
		t := true
		ty.CallFailed = &t
		return
	case 8: // Base
		t := true
		ty.InvalidDeposit = &t
		return
	case 9: // Base
		t := true
		ty.TransferFailed = &t
		return
	case 10: // Base
		t := true
		ty.MustCallByGov = &t
		return
	case 11: // Base
		t := true
		ty.PropNotOngoing = &t
		return
	case 12: // Base
		t := true
		ty.PropNotEnd = &t
		return
	case 13: // Base
		t := true
		ty.InvalidProposal = &t
		return
	case 14: // Base
		t := true
		ty.InvalidProposalStatus = &t
		return
	case 15: // Base
		t := true
		ty.InvalidProposalCaller = &t
		return
	case 16: // Base
		t := true
		ty.InvalidDepositTime = &t
		return
	case 17: // Base
		t := true
		ty.InvalidVoteTime = &t
		return
	case 18: // Base
		t := true
		ty.InvalidVoteStatus = &t
		return
	case 19: // Base
		t := true
		ty.InvalidVoteUser = &t
		return
	case 20: // Base
		t := true
		ty.ProposalInDecision = &t
		return
	case 21: // Base
		t := true
		ty.VoteAlreadyUnlocked = &t
		return
	case 22: // Base
		t := true
		ty.InvalidVoteUnlockTime = &t
		return
	case 23: // Base
		t := true
		ty.ProposalNotConfirmed = &t
		return
	case 24: // Base
		t := true
		ty.NoTrack = &t
		return
	case 25: // Base
		t := true
		ty.MaxBalanceOverflow = &t
		return
	case 26: // Base
		t := true
		ty.TransferDisable = &t
		return
	case 27: // Base
		t := true
		ty.InvalidVote = &t
		return
	case 28: // Base
		t := true
		ty.SetCodeFailed = &t
		return
	case 29: // Base
		t := true
		ty.SpendNotFound = &t
		return
	case 30: // Base
		t := true
		ty.SpendAlreadyExecuted = &t
		return
	case 31: // Base
		t := true
		ty.SpendTransferError = &t
		return
	case 32: // Base
		t := true
		ty.ReentrantCall = &t
		return
	default:
		return fmt.Errorf("unrecognized enum")
	}
}
func (ty *Error) Error() string {
	if ty.TokenNotFound != nil {
		return "TokenNotFound"
	}

	if ty.MemberExisted != nil {
		return "MemberExisted"
	}

	if ty.MemberNotExisted != nil {
		return "MemberNotExisted"
	}

	if ty.MemberBalanceNotZero != nil {
		return "MemberBalanceNotZero"
	}

	if ty.PublicJoinNotAllowed != nil {
		return "PublicJoinNotAllowed"
	}

	if ty.LowBalance != nil {
		return "LowBalance"
	}

	if ty.InsufficientAllowance != nil {
		return "InsufficientAllowance"
	}

	if ty.CallFailed != nil {
		return "CallFailed"
	}

	if ty.InvalidDeposit != nil {
		return "InvalidDeposit"
	}

	if ty.TransferFailed != nil {
		return "TransferFailed"
	}

	if ty.MustCallByGov != nil {
		return "MustCallByGov"
	}

	if ty.PropNotOngoing != nil {
		return "PropNotOngoing"
	}

	if ty.PropNotEnd != nil {
		return "PropNotEnd"
	}

	if ty.InvalidProposal != nil {
		return "InvalidProposal"
	}

	if ty.InvalidProposalStatus != nil {
		return "InvalidProposalStatus"
	}

	if ty.InvalidProposalCaller != nil {
		return "InvalidProposalCaller"
	}

	if ty.InvalidDepositTime != nil {
		return "InvalidDepositTime"
	}

	if ty.InvalidVoteTime != nil {
		return "InvalidVoteTime"
	}

	if ty.InvalidVoteStatus != nil {
		return "InvalidVoteStatus"
	}

	if ty.InvalidVoteUser != nil {
		return "InvalidVoteUser"
	}

	if ty.ProposalInDecision != nil {
		return "ProposalInDecision"
	}

	if ty.VoteAlreadyUnlocked != nil {
		return "VoteAlreadyUnlocked"
	}

	if ty.InvalidVoteUnlockTime != nil {
		return "InvalidVoteUnlockTime"
	}

	if ty.ProposalNotConfirmed != nil {
		return "ProposalNotConfirmed"
	}

	if ty.NoTrack != nil {
		return "NoTrack"
	}

	if ty.MaxBalanceOverflow != nil {
		return "MaxBalanceOverflow"
	}

	if ty.TransferDisable != nil {
		return "TransferDisable"
	}

	if ty.InvalidVote != nil {
		return "InvalidVote"
	}

	if ty.SetCodeFailed != nil {
		return "SetCodeFailed"
	}

	if ty.SpendNotFound != nil {
		return "SpendNotFound"
	}

	if ty.SpendAlreadyExecuted != nil {
		return "SpendAlreadyExecuted"
	}

	if ty.SpendTransferError != nil {
		return "SpendTransferError"
	}

	if ty.ReentrantCall != nil {
		return "ReentrantCall"
	}
	return "Unknown"
}

type Call struct { // Composite
	Contract     util.Option[types.H160]
	Selector     [4]byte
	Input        []byte
	Amount       types.U256
	RefTimeLimit uint64
	AllowReentry bool
}
type Curve struct { // Enum
	LinearDecreasing *struct { // 0
		F0 uint32
		F1 uint32
		F2 uint32
	}
	SteppedDecreasing *struct { // 1
		F0 uint32
		F1 uint32
		F2 uint32
		F3 uint32
	}
	Reciprocal *struct { // 2
		F0 uint32
		F1 uint32
		F2 int64
		F3 int64
	}
}

func (ty Curve) Encode(encoder scale.Encoder) (err error) {
	if ty.LinearDecreasing != nil {
		err = encoder.PushByte(0)
		if err != nil {
			return err
		}

		err = encoder.Encode(ty.LinearDecreasing.F0)
		if err != nil {
			return err
		}

		err = encoder.Encode(ty.LinearDecreasing.F1)
		if err != nil {
			return err
		}

		err = encoder.Encode(ty.LinearDecreasing.F2)
		if err != nil {
			return err
		}

		return nil
	}

	if ty.SteppedDecreasing != nil {
		err = encoder.PushByte(1)
		if err != nil {
			return err
		}

		err = encoder.Encode(ty.SteppedDecreasing.F0)
		if err != nil {
			return err
		}

		err = encoder.Encode(ty.SteppedDecreasing.F1)
		if err != nil {
			return err
		}

		err = encoder.Encode(ty.SteppedDecreasing.F2)
		if err != nil {
			return err
		}

		err = encoder.Encode(ty.SteppedDecreasing.F3)
		if err != nil {
			return err
		}

		return nil
	}

	if ty.Reciprocal != nil {
		err = encoder.PushByte(2)
		if err != nil {
			return err
		}

		err = encoder.Encode(ty.Reciprocal.F0)
		if err != nil {
			return err
		}

		err = encoder.Encode(ty.Reciprocal.F1)
		if err != nil {
			return err
		}

		err = encoder.Encode(ty.Reciprocal.F2)
		if err != nil {
			return err
		}

		err = encoder.Encode(ty.Reciprocal.F3)
		if err != nil {
			return err
		}

		return nil
	}
	return fmt.Errorf("unrecognized enum")
}

func (ty *Curve) Decode(decoder scale.Decoder) (err error) {
	variant, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}
	switch variant {
	case 0: // Tuple
		ty.LinearDecreasing = &struct {
			F0 uint32
			F1 uint32
			F2 uint32
		}{}

		err = decoder.Decode(&ty.LinearDecreasing.F0)
		if err != nil {
			return err
		}

		err = decoder.Decode(&ty.LinearDecreasing.F1)
		if err != nil {
			return err
		}

		err = decoder.Decode(&ty.LinearDecreasing.F2)
		if err != nil {
			return err
		}

		return
	case 1: // Tuple
		ty.SteppedDecreasing = &struct {
			F0 uint32
			F1 uint32
			F2 uint32
			F3 uint32
		}{}

		err = decoder.Decode(&ty.SteppedDecreasing.F0)
		if err != nil {
			return err
		}

		err = decoder.Decode(&ty.SteppedDecreasing.F1)
		if err != nil {
			return err
		}

		err = decoder.Decode(&ty.SteppedDecreasing.F2)
		if err != nil {
			return err
		}

		err = decoder.Decode(&ty.SteppedDecreasing.F3)
		if err != nil {
			return err
		}

		return
	case 2: // Tuple
		ty.Reciprocal = &struct {
			F0 uint32
			F1 uint32
			F2 int64
			F3 int64
		}{}

		err = decoder.Decode(&ty.Reciprocal.F0)
		if err != nil {
			return err
		}

		err = decoder.Decode(&ty.Reciprocal.F1)
		if err != nil {
			return err
		}

		err = decoder.Decode(&ty.Reciprocal.F2)
		if err != nil {
			return err
		}

		err = decoder.Decode(&ty.Reciprocal.F3)
		if err != nil {
			return err
		}

		return
	default:
		return fmt.Errorf("unrecognized enum")
	}
}

type Track struct { // Composite
	Name               []byte
	PreparePeriod      uint32
	DecisionDeposit    types.U256
	MaxDeciding        uint32
	ConfirmPeriod      uint32
	DecisionPeriod     uint32
	MinEnactmentPeriod uint32
	MaxBalance         types.U256
	MinApproval        Curve
	MinSupport         Curve
}
type Tuple_34 struct { // Tuple
	F0 uint16
	F1 Track
}
type TokenInfo struct { // Composite
	Name     []byte
	Symbol   []byte
	Decimals byte
}
