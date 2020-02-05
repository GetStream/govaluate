package govaluate

import (
	"bytes"
)

/*
	Holds a series of "transactions" which represent each token as it is output by an outputter (such as ToSQLQuery()).
	Some outputs (such as SQL) require a function call or non-c-like syntax to represent an expression.
	To accomplish this, this struct keeps track of each translated token as it is output, and can return and rollback those transactions.
*/
type expressionOutputStream struct {
	transactions []string
}

func (s *expressionOutputStream) add(transaction string) {
	s.transactions = append(s.transactions, transaction)
}

func (s *expressionOutputStream) rollback() string {

	index := len(s.transactions) - 1
	ret := s.transactions[index]

	s.transactions = s.transactions[:index]
	return ret
}

func (s *expressionOutputStream) createString(delimiter string) string {

	var retBuffer bytes.Buffer
	var transaction string

	penultimate := len(s.transactions) - 1

	for i := 0; i < penultimate; i++ {
		transaction = s.transactions[i]
		retBuffer.WriteString(transaction)
		retBuffer.WriteString(delimiter)
	}
	retBuffer.WriteString(s.transactions[penultimate])

	return retBuffer.String()
}
