package pod

import (
	"fmt"

	"github.com/centrifuge/go-substrate-rpc-client/v4/scale"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

type Error struct { // Enum
	SetCodeFailed           *bool // 0
	MustCallByCloudContract *bool // 1
	InsufficientBalance     *bool // 2
	PayFailed               *bool // 3
	NotOwner                *bool // 4
	NotEnoughAllowance      *bool // 5
	NotEnoughBalance        *bool // 6
	InvalidSideChainCaller  *bool // 7
	UnsupportedAsset        *bool // 8
	CodeUpgradeNotSupported *bool // 9
	AlreadyInitialized      *bool // 10
	NotSettled              *bool // 11
	CallFailed              *bool // 12
}

func (ty Error) Encode(encoder scale.Encoder) (err error) {
	if ty.SetCodeFailed != nil {
		err = encoder.PushByte(0)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.MustCallByCloudContract != nil {
		err = encoder.PushByte(1)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.InsufficientBalance != nil {
		err = encoder.PushByte(2)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.PayFailed != nil {
		err = encoder.PushByte(3)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.NotOwner != nil {
		err = encoder.PushByte(4)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.NotEnoughAllowance != nil {
		err = encoder.PushByte(5)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.NotEnoughBalance != nil {
		err = encoder.PushByte(6)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.InvalidSideChainCaller != nil {
		err = encoder.PushByte(7)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.UnsupportedAsset != nil {
		err = encoder.PushByte(8)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.CodeUpgradeNotSupported != nil {
		err = encoder.PushByte(9)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.AlreadyInitialized != nil {
		err = encoder.PushByte(10)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.NotSettled != nil {
		err = encoder.PushByte(11)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.CallFailed != nil {
		err = encoder.PushByte(12)
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
		ty.SetCodeFailed = &t
		return
	case 1: // Base
		t := true
		ty.MustCallByCloudContract = &t
		return
	case 2: // Base
		t := true
		ty.InsufficientBalance = &t
		return
	case 3: // Base
		t := true
		ty.PayFailed = &t
		return
	case 4: // Base
		t := true
		ty.NotOwner = &t
		return
	case 5: // Base
		t := true
		ty.NotEnoughAllowance = &t
		return
	case 6: // Base
		t := true
		ty.NotEnoughBalance = &t
		return
	case 7: // Base
		t := true
		ty.InvalidSideChainCaller = &t
		return
	case 8: // Base
		t := true
		ty.UnsupportedAsset = &t
		return
	case 9: // Base
		t := true
		ty.CodeUpgradeNotSupported = &t
		return
	case 10: // Base
		t := true
		ty.AlreadyInitialized = &t
		return
	case 11: // Base
		t := true
		ty.NotSettled = &t
		return
	case 12: // Base
		t := true
		ty.CallFailed = &t
		return
	default:
		return fmt.Errorf("unrecognized enum")
	}
}
func (ty *Error) Error() string {
	if ty.SetCodeFailed != nil {
		return "SetCodeFailed"
	}

	if ty.MustCallByCloudContract != nil {
		return "MustCallByCloudContract"
	}

	if ty.InsufficientBalance != nil {
		return "InsufficientBalance"
	}

	if ty.PayFailed != nil {
		return "PayFailed"
	}

	if ty.NotOwner != nil {
		return "NotOwner"
	}

	if ty.NotEnoughAllowance != nil {
		return "NotEnoughAllowance"
	}

	if ty.NotEnoughBalance != nil {
		return "NotEnoughBalance"
	}

	if ty.InvalidSideChainCaller != nil {
		return "InvalidSideChainCaller"
	}

	if ty.UnsupportedAsset != nil {
		return "UnsupportedAsset"
	}

	if ty.CodeUpgradeNotSupported != nil {
		return "CodeUpgradeNotSupported"
	}

	if ty.AlreadyInitialized != nil {
		return "AlreadyInitialized"
	}

	if ty.NotSettled != nil {
		return "NotSettled"
	}

	if ty.CallFailed != nil {
		return "CallFailed"
	}
	return "Unknown"
}

type AssetInfo struct { // Enum
	Native *[]byte   // 0
	ERC20  *struct { // 1
		F0 []byte
		F1 types.H256
	}
}

func (ty AssetInfo) Encode(encoder scale.Encoder) (err error) {
	if ty.Native != nil {
		err = encoder.PushByte(0)
		if err != nil {
			return err
		}
		err = encoder.Encode(*ty.Native)
		if err != nil {
			return err
		}
		return nil
	}

	if ty.ERC20 != nil {
		err = encoder.PushByte(1)
		if err != nil {
			return err
		}

		err = encoder.Encode(ty.ERC20.F0)
		if err != nil {
			return err
		}

		err = encoder.Encode(ty.ERC20.F1)
		if err != nil {
			return err
		}

		return nil
	}
	return fmt.Errorf("unrecognized enum")
}

func (ty *AssetInfo) Decode(decoder scale.Decoder) (err error) {
	variant, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}
	switch variant {
	case 0: // Inline
		ty.Native = new([]byte)
		err = decoder.Decode(ty.Native)
		if err != nil {
			return err
		}
		return
	case 1: // Tuple
		ty.ERC20 = &struct {
			F0 []byte
			F1 types.H256
		}{}

		err = decoder.Decode(&ty.ERC20.F0)
		if err != nil {
			return err
		}

		err = decoder.Decode(&ty.ERC20.F1)
		if err != nil {
			return err
		}

		return
	default:
		return fmt.Errorf("unrecognized enum")
	}
}
