package transaction

import (
	"encoding/json"

	"github.com/btcsuite/btcutil/base58"
	"github.com/stellar/go/keypair"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
)

// TODO versioning

type Transaction struct {
	T string
	H TransactionHeader
	B TransactionBody
}

type TransactionFromJSON struct {
	T string
	H TransactionHeader
	B TransactionBodyFromJSON
}

type TransactionBodyFromJSON struct {
	Source     string              `json:"source"`
	Fee        common.Amount       `json:"fee"`
	SequenceID uint64              `json:"sequenceid"`
	Operations []OperationFromJSON `json:"operations"`
}

func NewTransactionFromJSON(b []byte) (tx Transaction, err error) {
	var txt TransactionFromJSON
	if err = json.Unmarshal(b, &txt); err != nil {
		return
	}

	var operations []Operation
	for _, o := range txt.B.Operations {
		var op Operation
		if op, err = NewOperationFromInterface(o); err != nil {
			return
		}
		operations = append(operations, op)
	}

	tx.T = txt.T
	tx.H = txt.H
	tx.B = TransactionBody{
		Source:     txt.B.Source,
		Fee:        txt.B.Fee,
		SequenceID: txt.B.SequenceID,
		Operations: operations,
	}

	return
}

func NewTransaction(source string, sequenceID uint64, ops ...Operation) (tx Transaction, err error) {
	if len(ops) < 1 {
		err = errors.ErrorTransactionEmptyOperations
		return
	}

	txBody := TransactionBody{
		Source:     source,
		Fee:        common.BaseFee,
		SequenceID: sequenceID,
		Operations: ops,
	}

	tx = Transaction{
		T: "transaction",
		H: TransactionHeader{
			Created: common.NowISO8601(),
			Hash:    txBody.MakeHashString(),
		},
		B: txBody,
	}

	return
}

var TransactionWellFormedCheckerFuncs = []common.CheckerFunc{
	CheckTransactionSequenceID,
	CheckTransactionSource,
	CheckTransactionBaseFee,
	CheckTransactionOperation,
	CheckTransactionVerifySignature,
	CheckTransactionHashMatch,
}

func (tx Transaction) IsWellFormed(networkID []byte) (err error) {
	// TODO check `Version` format with SemVer

	checker := &TransactionChecker{
		DefaultChecker: common.DefaultChecker{Funcs: TransactionWellFormedCheckerFuncs},
		NetworkID:      networkID,
		Transaction:    tx,
	}
	if err = common.RunChecker(checker, common.DefaultDeferFunc); err != nil {
		return
	}

	return
}

func (tx Transaction) GetType() string {
	return tx.T
}

func (tx Transaction) Equal(m common.Message) bool {
	return tx.H.Hash == m.GetHash()
}

func (tx Transaction) IsValidSequenceID(sequenceID uint64) bool {
	return tx.B.SequenceID == sequenceID
}

func (tx Transaction) GetHash() string {
	return tx.H.Hash
}

func (tx Transaction) Source() string {
	return tx.B.Source
}

//
// Returns:
//   the total monetary value of this transaction,
//   which is the sum of its operations,
//   optionally with fees
//
// Params:
//   withFee = If fee should be included in the total
//
func (tx Transaction) TotalAmount(withFee bool) common.Amount {
	// Note that the transaction shouldn't be constructed invalid
	// (the sum of its Operations should not exceed the maximum supply)
	var amount common.Amount
	for _, op := range tx.B.Operations {
		amount = amount.MustAdd(op.B.GetAmount())
	}

	// TODO: This isn't checked anywhere yet
	if withFee {
		amount = amount.MustAdd(tx.B.Fee.MustMult(len(tx.B.Operations)))
	}

	return amount
}

func (tx Transaction) Serialize() (encoded []byte, err error) {
	encoded, err = json.Marshal(tx)
	return
}

func (tx Transaction) String() string {
	encoded, _ := json.MarshalIndent(tx, "", "  ")
	return string(encoded)
}

func (tx *Transaction) Sign(kp keypair.KP, networkID []byte) {
	tx.H.Hash = tx.B.MakeHashString()
	signature, _ := common.MakeSignature(kp, networkID, tx.H.Hash)

	tx.H.Signature = base58.Encode(signature)

	return
}

// NextSourceSequenceID returns the next sequenceID from the current Transaction.
//
// The sequenceID is simply the hash of the last paid transaction.
// It is present to prevent replay attacks.
func (tx Transaction) NextSequenceID() uint64 {
	return tx.B.SequenceID + 1
}

type TransactionHeader struct {
	Version   string `json:"version"`
	Created   string `json:"created"`
	Hash      string `json:"hash"`
	Signature string `json:"signature"`
}

type TransactionBody struct {
	Source     string        `json:"source"`
	Fee        common.Amount `json:"fee"`
	SequenceID uint64        `json:"sequenceID"`
	Operations []Operation   `json:"operations"`
}

func (tb TransactionBody) MakeHash() []byte {
	return common.MustMakeObjectHash(tb)
}

func (tb TransactionBody) MakeHashString() string {
	return base58.Encode(tb.MakeHash())
}
