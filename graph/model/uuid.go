package model

import (
	"database/sql/driver"
	"fmt"
	"io"
	"github.com/google/uuid"
)

type MSSQLUUID struct {
	uuid.UUID
}

// Scan implements the sql.Scanner interface for reading from DB
// SQL Server uses mixed-endian byte order for uniqueidentifier
func (m *MSSQLUUID) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("invalid type for MSSQLUUID: %T", value)
	}

	if len(bytes) != 16 {
		return fmt.Errorf("invalid UUID byte length: %d", len(bytes))
	}

	// Convert from SQL Server's mixed-endian byte order to RFC 4122 byte order
	// SQL Server stores: time_low (LE), time_mid (LE), time_hi_version (LE), clock_seq_hi (BE), clock_seq_low (BE), node (BE)
	var rfc4122Bytes [16]byte
	
	// Reverse the first 4 bytes (time_low) for little-endian to big-endian conversion
	rfc4122Bytes[0] = bytes[3]
	rfc4122Bytes[1] = bytes[2]
	rfc4122Bytes[2] = bytes[1]
	rfc4122Bytes[3] = bytes[0]
	
	// Reverse the next 2 bytes (time_mid)
	rfc4122Bytes[4] = bytes[5]
	rfc4122Bytes[5] = bytes[4]
	
	// Reverse the next 2 bytes (time_hi_version)
	rfc4122Bytes[6] = bytes[7]
	rfc4122Bytes[7] = bytes[6]
	
	// Copy the remaining 8 bytes as-is (clock_seq and node)
	copy(rfc4122Bytes[8:], bytes[8:])

	u, err := uuid.FromBytes(rfc4122Bytes[:])
	if err != nil {
		return err
	}
	m.UUID = u
	return nil
}

// Value implements the driver.Valuer interface for writing to DB
// Converts UUID to SQL Server's mixed-endian byte order
func (m MSSQLUUID) Value() (driver.Value, error) {
	rfc4122Bytes, err := m.UUID.MarshalBinary()
	if err != nil {
		return nil, err
	}

	// Convert from RFC 4122 byte order to SQL Server's mixed-endian byte order
	var sqlServerBytes [16]byte
	
	// Reverse the first 4 bytes (time_low)
	sqlServerBytes[0] = rfc4122Bytes[3]
	sqlServerBytes[1] = rfc4122Bytes[2]
	sqlServerBytes[2] = rfc4122Bytes[1]
	sqlServerBytes[3] = rfc4122Bytes[0]
	
	// Reverse the next 2 bytes (time_mid)
	sqlServerBytes[4] = rfc4122Bytes[5]
	sqlServerBytes[5] = rfc4122Bytes[4]
	
	// Reverse the next 2 bytes (time_hi_version)
	sqlServerBytes[6] = rfc4122Bytes[7]
	sqlServerBytes[7] = rfc4122Bytes[6]
	
	// Copy the remaining 8 bytes as-is
	copy(sqlServerBytes[8:], rfc4122Bytes[8:])

	return sqlServerBytes[:], nil
}

// MarshalGQL and UnmarshalGQL tell gqlgen how to handle this type
func (m MSSQLUUID) MarshalGQL(w io.Writer) {
	fmt.Fprintf(w, "%q", m.String())
}

func (m *MSSQLUUID) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("UUID must be a string")
	}
	u, err := uuid.Parse(str)
	if err != nil {
		return err
	}
	m.UUID = u
	return nil
}

// NilMSSQLUUID returns a nil MSSQLUUID
func NilMSSQLUUID() MSSQLUUID {
	return MSSQLUUID{UUID: uuid.Nil}
}
