package zendia

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var uuidType = reflect.TypeOf(uuid.UUID{})

type uuidCodec struct{}

func (uc *uuidCodec) EncodeValue(_ bsoncodec.EncodeContext, vw bsonrw.ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != uuidType {
		return bsoncodec.ValueEncoderError{Name: "UUIDEncodeValue", Types: []reflect.Type{uuidType}, Received: val}
	}
	u := val.Interface().(uuid.UUID)
	return vw.WriteBinaryWithSubtype(u[:], bsontype.BinaryUUID)
}

func (uc *uuidCodec) DecodeValue(_ bsoncodec.DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != uuidType {
		return bsoncodec.ValueDecoderError{Name: "UUIDDecodeValue", Types: []reflect.Type{uuidType}, Received: val}
	}

	switch vr.Type() {
	case bsontype.Binary:
		data, subtype, err := vr.ReadBinary()
		if err != nil {
			return err
		}
		if subtype != bsontype.BinaryUUID && subtype != 0x04 {
			return fmt.Errorf("invalid binary subtype for UUID: %d", subtype)
		}
		if len(data) != 16 {
			return fmt.Errorf("UUID binary data must be 16 bytes, got %d", len(data))
		}
		u, err := uuid.FromBytes(data)
		if err != nil {
			return err
		}
		val.Set(reflect.ValueOf(u))
	case bsontype.String:
		str, err := vr.ReadString()
		if err != nil {
			return err
		}
		u, err := uuid.Parse(str)
		if err != nil {
			return fmt.Errorf("cannot parse UUID string: %w", err)
		}
		val.Set(reflect.ValueOf(u))
	case bsontype.Null:
		if err := vr.ReadNull(); err != nil {
			return err
		}
		val.Set(reflect.ValueOf(uuid.Nil))
	default:
		return fmt.Errorf("cannot decode %s into UUID", vr.Type())
	}

	return nil
}

// MongoConnectConfig configuração de conexão MongoDB
type MongoConnectConfig struct {
	URI                     string
	Database                string
	MaxPoolSize             uint64
	MinPoolSize             uint64
	MaxConnIdleTime         time.Duration
	ServerSelectionTimeout  time.Duration
}

// MongoConnect cria uma conexão MongoDB com suporte nativo a UUID v4 e ObjectID.
//
// Uso:
//
//	db, err := zendia.MongoConnect(ctx, zendia.MongoConnectConfig{
//	    URI:      "mongodb://localhost:27017",
//	    Database: "myapp",
//	})
func MongoConnect(ctx context.Context, cfg MongoConnectConfig) (*mongo.Database, error) {
	if cfg.MaxPoolSize == 0 {
		cfg.MaxPoolSize = 10
	}
	if cfg.MinPoolSize == 0 {
		cfg.MinPoolSize = 2
	}
	if cfg.MaxConnIdleTime == 0 {
		cfg.MaxConnIdleTime = 5 * time.Minute
	}
	if cfg.ServerSelectionTimeout == 0 {
		cfg.ServerSelectionTimeout = 5 * time.Second
	}

	codec := &uuidCodec{}
	reg := bson.NewRegistry()
	reg.RegisterTypeEncoder(uuidType, codec)
	reg.RegisterTypeDecoder(uuidType, codec)

	clientOpts := options.Client().
		ApplyURI(cfg.URI).
		SetRegistry(reg).
		SetMaxPoolSize(cfg.MaxPoolSize).
		SetMinPoolSize(cfg.MinPoolSize).
		SetMaxConnIdleTime(cfg.MaxConnIdleTime).
		SetServerSelectionTimeout(cfg.ServerSelectionTimeout)

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("mongodb connection failed: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("mongodb ping failed: %w", err)
	}

	return client.Database(cfg.Database), nil
}
