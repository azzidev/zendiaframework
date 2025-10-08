package zendia

import (
	"reflect"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// convertUUIDs converte campos uuid.UUID para primitive.Binary com subtype 4
func convertUUIDs(entity interface{}) bson.M {
	doc := bson.M{}
	val := reflect.ValueOf(entity)
	
	// Se for ponteiro, pega o valor
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	
	typ := val.Type()
	
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)
		
		// Pega o nome do campo BSON
		bsonTag := fieldType.Tag.Get("bson")
		if bsonTag == "" || bsonTag == "-" {
			continue
		}
		
		// Remove opções como omitempty
		fieldName := bsonTag
		for commaIdx := 0; commaIdx < len(bsonTag); commaIdx++ {
			if bsonTag[commaIdx] == ',' {
				fieldName = bsonTag[:commaIdx]
				break
			}
		}
		
		// Converte UUID para Binary subtype 4
		if field.Type() == reflect.TypeOf(uuid.UUID{}) {
			uuidVal := field.Interface().(uuid.UUID)
			if uuidVal != uuid.Nil {
				doc[fieldName] = primitive.Binary{
					Subtype: 4,
					Data:    uuidVal[:],
				}
			}
		} else {
			// Outros tipos mantém como estão
			doc[fieldName] = field.Interface()
		}
	}
	
	return doc
}