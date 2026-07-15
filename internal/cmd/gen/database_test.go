package gen

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConvSQLType(t *testing.T) {
	tests := map[string]string{
		"int":          "int64",
		"bigint(20)":   "int64",
		"tinyint(1)":   "int32",
		"smallint":     "int32",
		"double(10,2)": "float64",
		"varchar(255)": "string",
		"longtext":     "string",
		"timestamp":    "time.Time",
		"json":         "interface{}",
	}
	for sqlType, expected := range tests {
		t.Run(sqlType, func(t *testing.T) {
			require.Equal(t, expected, convSQLType(sqlType))
		})
	}
}

func TestConvSQLTag(t *testing.T) {
	indexes := []Index{
		{KeyName: "idx_email", ColumnName: "email", NonUnique: 0},
		{KeyName: "idx_name", ColumnName: "name", NonUnique: 1},
	}
	tests := []struct {
		name     string
		column   Column
		expected string
	}{
		{
			name:     "primary key",
			column:   Column{Field: "id", Type: "bigint", Null: "NO", Key: "PRI"},
			expected: `gorm:"column:id;type:bigint;not null;primary_key"`,
		},
		{
			name:     "unique index and default",
			column:   Column{Field: "email", Type: "varchar(255)", Null: "NO", Key: "UNI", Default: ""},
			expected: `gorm:"column:email;type:varchar(255);not null;index:idx_email,unique"`,
		},
		{
			name:     "non unique index",
			column:   Column{Field: "name", Type: "varchar(64)", Key: "MUL", Default: "guest"},
			expected: `gorm:"column:name;type:varchar(64);default:guest;index:idx_name"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, convSQLTag(tt.column, indexes))
		})
	}
}
