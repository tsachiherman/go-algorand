package txnsync

// Code generated by github.com/algorand/msgp DO NOT EDIT.

import (
	"github.com/algorand/msgp/msgp"
)

// The following msgp objects are implemented in this file:
// encodedBloomFilter
//          |-----> (*) MarshalMsg
//          |-----> (*) CanMarshalMsg
//          |-----> (*) UnmarshalMsg
//          |-----> (*) CanUnmarshalMsg
//          |-----> (*) Msgsize
//          |-----> (*) MsgIsZero
//
// packedTransactionGroups
//            |-----> (*) MarshalMsg
//            |-----> (*) CanMarshalMsg
//            |-----> (*) UnmarshalMsg
//            |-----> (*) CanUnmarshalMsg
//            |-----> (*) Msgsize
//            |-----> (*) MsgIsZero
//
// requestParams
//       |-----> (*) MarshalMsg
//       |-----> (*) CanMarshalMsg
//       |-----> (*) UnmarshalMsg
//       |-----> (*) CanUnmarshalMsg
//       |-----> (*) Msgsize
//       |-----> (*) MsgIsZero
//
// timingParams
//       |-----> (*) MarshalMsg
//       |-----> (*) CanMarshalMsg
//       |-----> (*) UnmarshalMsg
//       |-----> (*) CanUnmarshalMsg
//       |-----> (*) Msgsize
//       |-----> (*) MsgIsZero
//
// transactionBlockMessage
//            |-----> (*) MarshalMsg
//            |-----> (*) CanMarshalMsg
//            |-----> (*) UnmarshalMsg
//            |-----> (*) CanUnmarshalMsg
//            |-----> (*) Msgsize
//            |-----> (*) MsgIsZero
//

// MarshalMsg implements msgp.Marshaler
func (z *encodedBloomFilter) MarshalMsg(b []byte) (o []byte) {
	o = msgp.Require(b, z.Msgsize())
	// omitempty: check for empty values
	zb0001Len := uint32(0)
	// variable map header, size zb0001Len
	o = append(o, 0x80|uint8(zb0001Len))
	if zb0001Len != 0 {
	}
	return
}

func (_ *encodedBloomFilter) CanMarshalMsg(z interface{}) bool {
	_, ok := (z).(*encodedBloomFilter)
	return ok
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *encodedBloomFilter) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0001 int
	var zb0002 bool
	zb0001, zb0002, bts, err = msgp.ReadMapHeaderBytes(bts)
	if _, ok := err.(msgp.TypeError); ok {
		zb0001, zb0002, bts, err = msgp.ReadArrayHeaderBytes(bts)
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		if zb0001 > 0 {
			err = msgp.ErrTooManyArrayFields(zb0001)
			if err != nil {
				err = msgp.WrapError(err, "struct-from-array")
				return
			}
		}
	} else {
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		if zb0002 {
			(*z) = encodedBloomFilter{}
		}
		for zb0001 > 0 {
			zb0001--
			field, bts, err = msgp.ReadMapKeyZC(bts)
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
			switch string(field) {
			default:
				err = msgp.ErrNoField(string(field))
				if err != nil {
					err = msgp.WrapError(err)
					return
				}
			}
		}
	}
	o = bts
	return
}

func (_ *encodedBloomFilter) CanUnmarshalMsg(z interface{}) bool {
	_, ok := (z).(*encodedBloomFilter)
	return ok
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *encodedBloomFilter) Msgsize() (s int) {
	s = 1
	return
}

// MsgIsZero returns whether this is a zero value
func (z *encodedBloomFilter) MsgIsZero() bool {
	return true
}

// MarshalMsg implements msgp.Marshaler
func (z *packedTransactionGroups) MarshalMsg(b []byte) (o []byte) {
	o = msgp.Require(b, z.Msgsize())
	// omitempty: check for empty values
	zb0003Len := uint32(0)
	// variable map header, size zb0003Len
	o = append(o, 0x80|uint8(zb0003Len))
	if zb0003Len != 0 {
	}
	return
}

func (_ *packedTransactionGroups) CanMarshalMsg(z interface{}) bool {
	_, ok := (z).(*packedTransactionGroups)
	return ok
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *packedTransactionGroups) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0003 int
	var zb0004 bool
	zb0003, zb0004, bts, err = msgp.ReadMapHeaderBytes(bts)
	if _, ok := err.(msgp.TypeError); ok {
		zb0003, zb0004, bts, err = msgp.ReadArrayHeaderBytes(bts)
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		if zb0003 > 0 {
			err = msgp.ErrTooManyArrayFields(zb0003)
			if err != nil {
				err = msgp.WrapError(err, "struct-from-array")
				return
			}
		}
	} else {
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		if zb0004 {
			(*z) = packedTransactionGroups{}
		}
		for zb0003 > 0 {
			zb0003--
			field, bts, err = msgp.ReadMapKeyZC(bts)
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
			switch string(field) {
			default:
				err = msgp.ErrNoField(string(field))
				if err != nil {
					err = msgp.WrapError(err)
					return
				}
			}
		}
	}
	o = bts
	return
}

func (_ *packedTransactionGroups) CanUnmarshalMsg(z interface{}) bool {
	_, ok := (z).(*packedTransactionGroups)
	return ok
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *packedTransactionGroups) Msgsize() (s int) {
	s = 1
	return
}

// MsgIsZero returns whether this is a zero value
func (z *packedTransactionGroups) MsgIsZero() bool {
	return true
}

// MarshalMsg implements msgp.Marshaler
func (z *requestParams) MarshalMsg(b []byte) (o []byte) {
	o = msgp.Require(b, z.Msgsize())
	// omitempty: check for empty values
	zb0001Len := uint32(0)
	// variable map header, size zb0001Len
	o = append(o, 0x80|uint8(zb0001Len))
	if zb0001Len != 0 {
	}
	return
}

func (_ *requestParams) CanMarshalMsg(z interface{}) bool {
	_, ok := (z).(*requestParams)
	return ok
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *requestParams) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0001 int
	var zb0002 bool
	zb0001, zb0002, bts, err = msgp.ReadMapHeaderBytes(bts)
	if _, ok := err.(msgp.TypeError); ok {
		zb0001, zb0002, bts, err = msgp.ReadArrayHeaderBytes(bts)
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		if zb0001 > 0 {
			err = msgp.ErrTooManyArrayFields(zb0001)
			if err != nil {
				err = msgp.WrapError(err, "struct-from-array")
				return
			}
		}
	} else {
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		if zb0002 {
			(*z) = requestParams{}
		}
		for zb0001 > 0 {
			zb0001--
			field, bts, err = msgp.ReadMapKeyZC(bts)
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
			switch string(field) {
			default:
				err = msgp.ErrNoField(string(field))
				if err != nil {
					err = msgp.WrapError(err)
					return
				}
			}
		}
	}
	o = bts
	return
}

func (_ *requestParams) CanUnmarshalMsg(z interface{}) bool {
	_, ok := (z).(*requestParams)
	return ok
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *requestParams) Msgsize() (s int) {
	s = 1
	return
}

// MsgIsZero returns whether this is a zero value
func (z *requestParams) MsgIsZero() bool {
	return true
}

// MarshalMsg implements msgp.Marshaler
func (z *timingParams) MarshalMsg(b []byte) (o []byte) {
	o = msgp.Require(b, z.Msgsize())
	// omitempty: check for empty values
	zb0002Len := uint32(0)
	// variable map header, size zb0002Len
	o = append(o, 0x80|uint8(zb0002Len))
	if zb0002Len != 0 {
	}
	return
}

func (_ *timingParams) CanMarshalMsg(z interface{}) bool {
	_, ok := (z).(*timingParams)
	return ok
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *timingParams) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0002 int
	var zb0003 bool
	zb0002, zb0003, bts, err = msgp.ReadMapHeaderBytes(bts)
	if _, ok := err.(msgp.TypeError); ok {
		zb0002, zb0003, bts, err = msgp.ReadArrayHeaderBytes(bts)
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		if zb0002 > 0 {
			err = msgp.ErrTooManyArrayFields(zb0002)
			if err != nil {
				err = msgp.WrapError(err, "struct-from-array")
				return
			}
		}
	} else {
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		if zb0003 {
			(*z) = timingParams{}
		}
		for zb0002 > 0 {
			zb0002--
			field, bts, err = msgp.ReadMapKeyZC(bts)
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
			switch string(field) {
			default:
				err = msgp.ErrNoField(string(field))
				if err != nil {
					err = msgp.WrapError(err)
					return
				}
			}
		}
	}
	o = bts
	return
}

func (_ *timingParams) CanUnmarshalMsg(z interface{}) bool {
	_, ok := (z).(*timingParams)
	return ok
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *timingParams) Msgsize() (s int) {
	s = 1
	return
}

// MsgIsZero returns whether this is a zero value
func (z *timingParams) MsgIsZero() bool {
	return true
}

// MarshalMsg implements msgp.Marshaler
func (z *transactionBlockMessage) MarshalMsg(b []byte) (o []byte) {
	o = msgp.Require(b, z.Msgsize())
	// omitempty: check for empty values
	zb0001Len := uint32(0)
	// variable map header, size zb0001Len
	o = append(o, 0x80|uint8(zb0001Len))
	if zb0001Len != 0 {
	}
	return
}

func (_ *transactionBlockMessage) CanMarshalMsg(z interface{}) bool {
	_, ok := (z).(*transactionBlockMessage)
	return ok
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *transactionBlockMessage) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0001 int
	var zb0002 bool
	zb0001, zb0002, bts, err = msgp.ReadMapHeaderBytes(bts)
	if _, ok := err.(msgp.TypeError); ok {
		zb0001, zb0002, bts, err = msgp.ReadArrayHeaderBytes(bts)
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		if zb0001 > 0 {
			err = msgp.ErrTooManyArrayFields(zb0001)
			if err != nil {
				err = msgp.WrapError(err, "struct-from-array")
				return
			}
		}
	} else {
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		if zb0002 {
			(*z) = transactionBlockMessage{}
		}
		for zb0001 > 0 {
			zb0001--
			field, bts, err = msgp.ReadMapKeyZC(bts)
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
			switch string(field) {
			default:
				err = msgp.ErrNoField(string(field))
				if err != nil {
					err = msgp.WrapError(err)
					return
				}
			}
		}
	}
	o = bts
	return
}

func (_ *transactionBlockMessage) CanUnmarshalMsg(z interface{}) bool {
	_, ok := (z).(*transactionBlockMessage)
	return ok
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *transactionBlockMessage) Msgsize() (s int) {
	s = 1
	return
}

// MsgIsZero returns whether this is a zero value
func (z *transactionBlockMessage) MsgIsZero() bool {
	return true
}
