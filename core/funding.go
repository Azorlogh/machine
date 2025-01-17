package core

import (
	"errors"
	"fmt"
)

type FundingPart struct {
	Amount  uint64
	Account Account
}

type Funding struct {
	Asset    Asset
	Parts    []FundingPart
	Infinite bool
}

func (lhs *Funding) Equals(rhs *Funding) bool {
	if lhs.Asset != rhs.Asset {
		return false
	}
	if len(lhs.Parts) != len(rhs.Parts) {
		return false
	}
	for i := range lhs.Parts {
		if lhs.Parts[i] != rhs.Parts[i] {
			return false
		}
	}
	return true
}

func (f Funding) String() string {
	out := fmt.Sprintf("[%v", string(f.Asset))
	for _, part := range f.Parts {
		out += fmt.Sprintf(" %v %v", part.Account, part.Amount)
	}
	if f.Infinite {
		out += " @world *"
	}
	return out + "]"
}

func (f Funding) Take(amount uint64) (Funding, Funding, error) {
	result := Funding{
		Asset: f.Asset,
	}
	remainder := Funding{
		Asset: f.Asset,
	}
	remaining_to_withdraw := amount
	i := 0
	for remaining_to_withdraw > 0 && i < len(f.Parts) {
		amt_to_withdraw := f.Parts[i].Amount
		// if this part has excess balance, put it in the remainder & only take what's needed
		if amt_to_withdraw > remaining_to_withdraw {
			rem := amt_to_withdraw - remaining_to_withdraw
			amt_to_withdraw = remaining_to_withdraw
			remainder.Parts = append(remainder.Parts, FundingPart{
				Account: f.Parts[i].Account,
				Amount:  rem,
			})
		}
		remaining_to_withdraw -= amt_to_withdraw
		result.Parts = append(result.Parts, FundingPart{
			Account: f.Parts[i].Account,
			Amount:  amt_to_withdraw,
		})
		i++
	}
	for i < len(f.Parts) {
		remainder.Parts = append(remainder.Parts, FundingPart{
			Account: f.Parts[i].Account,
			Amount:  f.Parts[i].Amount,
		})
		i++
	}
	if f.Infinite {
		remainder.Infinite = true
	}
	if remaining_to_withdraw != 0 {
		if f.Infinite {
			result.Parts = append(result.Parts, FundingPart{
				Account: "world",
				Amount:  remaining_to_withdraw,
			})
		} else {
			return Funding{}, Funding{}, errors.New("insufficient funding")
		}
	}
	return result, remainder, nil
}

func (f Funding) TakeMax(amount uint64) (Funding, Funding) {
	result := Funding{
		Asset: f.Asset,
	}
	remainder := Funding{
		Asset: f.Asset,
	}
	remaining_to_withdraw := amount
	i := 0
	for remaining_to_withdraw > 0 && i < len(f.Parts) {
		amt_to_withdraw := f.Parts[i].Amount
		// if this part has excess balance, put it in the remainder & only take what's needed
		if amt_to_withdraw > remaining_to_withdraw {
			rem := amt_to_withdraw - remaining_to_withdraw
			amt_to_withdraw = remaining_to_withdraw
			remainder.Parts = append(remainder.Parts, FundingPart{
				Account: f.Parts[i].Account,
				Amount:  rem,
			})
		}
		remaining_to_withdraw -= amt_to_withdraw
		result.Parts = append(result.Parts, FundingPart{
			Account: f.Parts[i].Account,
			Amount:  amt_to_withdraw,
		})
		i++
	}
	for i < len(f.Parts) {
		remainder.Parts = append(remainder.Parts, FundingPart{
			Account: f.Parts[i].Account,
			Amount:  f.Parts[i].Amount,
		})
		i++
	}
	if f.Infinite {
		remainder.Infinite = true
	}
	if remaining_to_withdraw != 0 && f.Infinite {
		result.Parts = append(result.Parts, FundingPart{
			Account: "world",
			Amount:  remaining_to_withdraw,
		})
	}
	return result, remainder
}

func (f Funding) Concat(other Funding) (Funding, error) {
	if f.Asset != other.Asset {
		return Funding{}, errors.New("tried to concat different assets")
	}
	res := Funding{
		Asset:    f.Asset,
		Parts:    f.Parts,
		Infinite: f.Infinite || other.Infinite,
	}
	if !f.Infinite {
		if len(res.Parts) > 0 && len(other.Parts) > 0 && res.Parts[len(res.Parts)-1].Account == other.Parts[0].Account {
			res.Parts[len(res.Parts)-1].Amount += other.Parts[0].Amount
			res.Parts = append(res.Parts, other.Parts[1:]...)
		} else {
			res.Parts = append(res.Parts, other.Parts...)
		}
	}
	return res, nil
}

func (f Funding) Total() (uint64, error) {
	if f.Infinite {
		return 0, errors.New("tried to calculate total of infinite funding")
	}
	total := uint64(0)
	for _, part := range f.Parts {
		total += part.Amount
	}
	return total, nil
}

func (f Funding) Reverse() (*Funding, error) {
	if f.Infinite {
		return nil, errors.New("tried to reverse an infinite funding")
	}
	new_parts := []FundingPart{}
	for i := len(f.Parts) - 1; i >= 0; i-- {
		new_parts = append(new_parts, f.Parts[i])
	}
	new_funding := Funding{
		Asset:    f.Asset,
		Parts:    new_parts,
		Infinite: false,
	}
	return &new_funding, nil
}
