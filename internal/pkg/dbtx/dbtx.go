// Package dbtx defines the minimal transaction surface exposed to
// cascade-delete callbacks. It keeps persistence types out of transport
// contracts (internal/client) so the client layer can later map to gRPC
// without importing gorm.
package dbtx

import "gorm.io/gorm"

// Tx is the subset of *gorm.DB methods used by cascade-delete callbacks.
// gorm.DB satisfies it structurally; service providers pass their *gorm.DB
// transaction directly.
type Tx interface {
	Model(value interface{}) *gorm.DB
	Where(query interface{}, args ...interface{}) *gorm.DB
	Create(value interface{}) *gorm.DB
	First(dest interface{}, conds ...interface{}) *gorm.DB
	Pluck(column string, dest interface{}) *gorm.DB
	Find(dest interface{}, conds ...interface{}) *gorm.DB
	Delete(value interface{}, conds ...interface{}) *gorm.DB
	Updates(values interface{}) *gorm.DB
}

// Compile-time proof that gorm.DB implements Tx.
var _ Tx = (*gorm.DB)(nil)
