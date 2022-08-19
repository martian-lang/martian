// Copyright (c) 2022 10X Genomics, Inc. All rights reserved.

package core

type atomicWriteErrorWrapper struct {
	Err error
	Msg string
}

func wrapAtomicError(msg string, err error) error {
	if err == nil {
		return nil
	}
	return &atomicWriteErrorWrapper{
		Msg: msg,
		Err: err,
	}
}

func (err *atomicWriteErrorWrapper) Error() string {
	return err.Msg + ": " + err.Err.Error()
}

func (err *atomicWriteErrorWrapper) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.Err
}
