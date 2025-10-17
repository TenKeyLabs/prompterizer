package prompterizer_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/shopspring/decimal"
	"github.com/tenkeylabs/prompterizer"
	"google.golang.org/genai"
)

type Metadata struct {
	Location string `json:"type" prompt:"location,string"`
}

type Event struct {
	Name string `json:"name" prompt:"name,string" prompt_description:"The name of the event"`
}

type Embedded struct {
	EmbeddedField string `json:"embeddedField" prompt:"embeddedField,string,required"`
}

type TestPrompt struct {
	unexported     string
	Ignored        string          `json:"ignored"`
	DocumentDate   time.Time       `json:"documentDate" prompt:"documentDate,string,required"`
	CreationDate   time.Time       `json:"creationDate" prompt:"creationDate,string,date-time,required"`
	PublishDate    string          `json:"publishDate" prompt:"publishDate,string" prompt_description:"The date the document was published"`
	Title          string          `json:"title" prompt:"title,string" prompt_description:"The title of the document under the series {seriesName}"`
	FirstName      string          `json:"firstName" prompt:"firstName,string" prompt_aliases:"givenName"`
	LastName       string          `json:"lastName" prompt:"lastName,string" prompt_aliases:"surName,familyName"`
	Witness        string          `json:"witness" prompt:"witness,string" prompt_description:"The person from {seriesName} who witnessed the document signing." prompt_aliases:"witnessName,witnessedBy"`
	IsParsed       bool            `json:"isParsed" prompt:"isParsed,bool"`
	Count          int             `json:"count" prompt:"count,integer"`
	Status         string          `json:"status" prompt:"status,string" prompt_enum:"active,inactive,pending"`
	StatusCode     int             `json:"statusCode" prompt:"statusCode,integer,httpStatus" prompt_enum:"200,400,500" prompt_description:"HTTP status code for the document"`
	Percentage     float64         `json:"percentage" prompt:"percentage,number"`
	Amount         decimal.Decimal `json:"amount" prompt:"amount,number"`
	Metadata       Metadata        `json:"metadata" prompt:"metadata,object"`
	Tags           []string        `json:"tags" prompt:"tags,string"`
	Events         []Event         `json:"events" prompt:"events,object"`
	SpecialEvent   *Event          `json:"specialEvent" prompt:"specialEvent,object"`
	OptionalEvents []*Event        `json:"optionalEvents" prompt:"optionalEvents,object"`
	TagSets        [][]string      `json:"tagSets" prompt:"tagSets,string"`
	Embedded
}

type InvalidType struct {
	Field string `json:"invalidField" prompt:"invalidField,"`
}

type TypeMismatch struct {
	Field string `json:"mismatchedField" prompt:"mismatchedField,integer"`
}

type WrongNumberOfParams struct {
	Field string `json:"wrongNumberOfParams" prompt:"wrongNumberOfParams"`
}

type UnsupportedType struct {
	Field complex64 `json:"unsupportedField" prompt:"unsupportedField,integer"`
}

var _ = Describe("Transform", func() {
	Describe("MarshalResponseSchema", func() {
		var schema *genai.Schema

		BeforeEach(func() {
			var err error
			schema, err = prompterizer.MarshalResponseSchema(TestPrompt{}, map[string]string{"seriesName": "Business 101"})
			Expect(err).ToNot(HaveOccurred())
		})

		Context("success", func() {
			It("should succeed with a pointer to a struct", func() {
				schema, err := prompterizer.MarshalResponseSchema(&TestPrompt{}, map[string]string{"seriesName": "Business 101"})
				Expect(err).ToNot(HaveOccurred())
				Expect(schema).NotTo(BeNil())
			})

			It("should ignore unexported fields", func() {
				_ = TestPrompt{}.unexported // to avoid unused variable lint error
				Expect(schema.Properties).ToNot(HaveKey("unexported"))
			})

			It("should ignore fields without a prompt tag", func() {
				Expect(schema.Properties).ToNot(HaveKey("ignored"))
			})

			It("should marshal a time.Time property", func() {
				Expect(schema).To(PointTo(MatchFields(IgnoreExtras, Fields{
					"Properties": HaveKey("documentDate"),
					"Required":   ContainElement("documentDate"),
				})))

				Expect(schema.Properties["documentDate"]).To(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type": Equal(genai.TypeString),
				})))
			})

			It("should marshal a property with explicit format if provided in prompt tag", func() {
				Expect(schema.Required).To(ContainElement("creationDate"))
				Expect(schema.Properties).To(HaveKey("creationDate"))
				Expect(schema.Properties["creationDate"]).To(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(genai.TypeString),
					"Format": Equal("date-time"),
				})))
			})

			It("should marshal a string property with a description", func() {
				Expect(schema.Properties).To(HaveKey("publishDate"))
				Expect(schema.Properties["publishDate"]).To(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":        Equal(genai.TypeString),
					"Description": Equal("The date the document was published"),
				})))
			})

			It("should marshal a string property with a templated description", func() {
				Expect(schema.Properties).To(HaveKey("title"))
				Expect(schema.Properties["title"]).To(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":        Equal(genai.TypeString),
					"Description": Equal("The title of the document under the series Business 101"),
				})))
			})

			It("should marshal a string property with an alias", func() {
				Expect(schema.Properties).To(HaveKey("firstName"))
				Expect(schema.Properties["firstName"]).To(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":        Equal(genai.TypeString),
					"Description": Equal("Also commonly reported as 'givenName'."),
				})))
			})

			It("should marshal a string property with aliases", func() {
				Expect(schema.Properties).To(HaveKey("lastName"))
				Expect(schema.Properties["lastName"]).To(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":        Equal(genai.TypeString),
					"Description": Equal("Also commonly reported as 'surName', 'familyName'."),
				})))
			})

			It("should marshal a string property with a templated description and aliases", func() {
				Expect(schema.Properties).To(HaveKey("witness"))
				Expect(schema.Properties["witness"]).To(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":        Equal(genai.TypeString),
					"Description": Equal("The person from Business 101 who witnessed the document signing. Also commonly reported as 'witnessName', 'witnessedBy'."),
				})))
			})

			It("should marshal a bool property", func() {
				Expect(schema.Properties).To(HaveKey("isParsed"))
				Expect(schema.Properties["isParsed"]).To(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type": Equal(genai.TypeBoolean),
				})))
			})

			It("should marshal an integer property", func() {
				Expect(schema.Properties).To(HaveKey("count"))
				Expect(schema.Properties["count"]).To(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type": Equal(genai.TypeInteger),
				})))
			})

			It("should marshal a property with enum values and enum format if prompt_enum is set", func() {
				Expect(schema.Properties).To(HaveKey("status"))

				Expect(schema.Properties["status"]).To(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(genai.TypeString),
					"Format": Equal("enum"),
					"Enum":   ConsistOf("active", "inactive", "pending"),
				})))
			})

			It("should marshal a property with the explicit format even if prompt_enum is set", func() {
				Expect(schema.Properties).To(HaveKey("statusCode"))

				Expect(schema.Properties["statusCode"]).To(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(genai.TypeInteger),
					"Format": Equal("httpStatus"),
					"Enum":   ConsistOf("200", "400", "500"),
				})))
			})

			It("should marshal a property with a float format for number type", func() {
				Expect(schema.Properties).To(HaveKey("percentage"))

				Expect(schema.Properties["percentage"]).To(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(genai.TypeNumber),
					"Format": Equal("float"),
				})))
			})

			It("should marshal a decimal.Decimal property", func() {
				Expect(schema.Properties).To(HaveKey("amount"))
				Expect(schema.Properties["amount"]).To(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type": Equal(genai.TypeNumber),
				})))
			})

			It("should marshal a nested struct", func() {
				Expect(schema.Properties).To(HaveKey("metadata"))
				Expect(schema.Properties["metadata"]).To(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type": Equal(genai.TypeObject),
					"Properties": MatchAllKeys(Keys{
						"location": PointTo(MatchFields(IgnoreExtras, Fields{
							"Type": Equal(genai.TypeString),
						})),
					}),
				})))
			})

			It("should marshal an array of objects", func() {
				Expect(schema.Properties).To(HaveKey("events"))
				Expect(schema.Properties["events"]).To(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type": Equal(genai.TypeArray),
					"Items": PointTo(MatchFields(IgnoreExtras, Fields{
						"Properties": MatchAllKeys(Keys{
							"name": PointTo(MatchFields(IgnoreExtras, Fields{
								"Type":        Equal(genai.TypeString),
								"Description": Equal("The name of the event"),
							})),
						}),
					})),
				})))
			})

			It("should marshal a pointer field as nullable", func() {
				Expect(schema.Properties).To(HaveKey("specialEvent"))

				Expect(schema.Properties["specialEvent"]).To(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(genai.TypeObject),
					"Nullable": PointTo(BeTrue()),
				})))
			})

			It("should marshal a slice of pointers", func() {
				Expect(schema.Properties).To(HaveKey("optionalEvents"))
				Expect(schema.Properties["optionalEvents"]).To(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type": Equal(genai.TypeArray),
					"Items": PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":     Equal(genai.TypeObject),
						"Nullable": PointTo(BeTrue()),
					})),
				})))
			})

			It("should marshal a nested array of strings", func() {
				Expect(schema.Properties).To(HaveKey("tagSets"))
				Expect(schema.Properties["tagSets"]).To(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type": Equal(genai.TypeArray),
					"Items": PointTo(MatchFields(IgnoreExtras, Fields{
						"Type": Equal(genai.TypeArray),
						"Items": PointTo(MatchFields(IgnoreExtras, Fields{
							"Type": Equal(genai.TypeString),
						})),
					})),
				})))
			})

			It("should marshal an embedded struct's fields", func() {
				Expect(schema.Required).To(ContainElement("embeddedField"))
				Expect(schema.Properties).To(HaveKey("embeddedField"))
				Expect(schema.Properties["embeddedField"]).To(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type": Equal(genai.TypeString),
				})))
			})

			It("should populate PropertyOrdering with the correct field order", func() {
				Expect(schema.PropertyOrdering).NotTo(BeEmpty())
				Expect(schema.PropertyOrdering).To(HaveLen(20))

				// Verify fields are in the order they appear in the struct
				// Note: In TestPrompt, Embedded is declared at the end of the struct
				expectedOrder := []string{
					"documentDate", // first field with prompt tag
					"creationDate", // second field with prompt tag
					"publishDate",  // third field with prompt tag
					"title",        // and so on...
					"firstName",
					"lastName",
					"witness",
					"isParsed",
					"count",
					"status",
					"statusCode",
					"percentage",
					"amount",
					"metadata",
					"tags",
					"events",
					"specialEvent",
					"optionalEvents",
					"tagSets",
					"embeddedField", // from Embedded struct (declared at end)
				}

				Expect(schema.PropertyOrdering).To(Equal(expectedOrder))

				// Also verify each property exists in the schema
				for _, propName := range expectedOrder {
					Expect(schema.Properties).To(HaveKey(propName),
						"Property '%s' should exist in schema", propName)
				}
			})

			It("should populate PropertyOrdering for nested objects", func() {
				Expect(schema.Properties).To(HaveKey("metadata"))
				metadataSchema := schema.Properties["metadata"]
				Expect(metadataSchema.PropertyOrdering).To(Equal([]string{"location"}))
			})

			It("should populate PropertyOrdering for array item schemas when they are objects", func() {
				Expect(schema.Properties).To(HaveKey("events"))
				eventsSchema := schema.Properties["events"]
				Expect(eventsSchema.Items).NotTo(BeNil())
				Expect(eventsSchema.Items.PropertyOrdering).To(Equal([]string{"name"}))
			})
		})

		Context("PropertyOrdering with simple struct", func() {
			type SimpleStruct struct {
				Zebra   string `prompt:"zebra,string"`
				Alpha   string `prompt:"alpha,string"`
				Bravo   string `prompt:"bravo,string"`
				Charlie string `prompt:"charlie,string"`
			}

			It("should preserve field order even if not alphabetical", func() {
				schema, err := prompterizer.MarshalResponseSchema(SimpleStruct{}, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(schema.PropertyOrdering).To(Equal([]string{"zebra", "alpha", "bravo", "charlie"}))
			})
		})

		Context("PropertyOrdering with embedded structs", func() {
			type Base struct {
				BaseField1 string `prompt:"baseField1,string"`
				BaseField2 string `prompt:"baseField2,string"`
			}

			type Extended struct {
				Base
				ExtField1 string `prompt:"extField1,string"`
				ExtField2 string `prompt:"extField2,string"`
			}

			It("should place embedded struct fields first, followed by the struct's own fields", func() {
				schema, err := prompterizer.MarshalResponseSchema(Extended{}, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(schema.PropertyOrdering).To(Equal([]string{
					"baseField1",
					"baseField2",
					"extField1",
					"extField2",
				}))
			})
		})

		Context("PropertyOrdering with multiple embedded structs", func() {
			type EmbedOne struct {
				One string `prompt:"one,string"`
			}

			type EmbedTwo struct {
				Two string `prompt:"two,string"`
			}

			type Combined struct {
				EmbedOne
				Middle string `prompt:"middle,string"`
				EmbedTwo
				Last string `prompt:"last,string"`
			}

			It("should preserve the order of embedded structs as they appear", func() {
				schema, err := prompterizer.MarshalResponseSchema(Combined{}, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(schema.PropertyOrdering).To(Equal([]string{
					"one",
					"middle",
					"two",
					"last",
				}))
			})
		})

		Context("PropertyOrdering with fields without prompt tags", func() {
			type Mixed struct {
				First  string `prompt:"first,string"`
				Second string // no prompt tag
				Third  string `prompt:"third,string"`
				fourth string `prompt:"fourth,string"` // unexported
				Fifth  string `prompt:"fifth,string"`
			}

			It("should only include fields with prompt tags", func() {
				schema, err := prompterizer.MarshalResponseSchema(Mixed{}, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(schema.PropertyOrdering).To(Equal([]string{
					"first",
					"third",
					"fifth",
				}))
			})
		})

		Context("errors", func() {
			It("should return an error if the value is nil", func() {
				_, err := prompterizer.MarshalResponseSchema(nil, map[string]string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("input value for schema generation cannot be nil"))
			})

			It("should return an error if the value is not a struct", func() {
				_, err := prompterizer.MarshalResponseSchema("not a struct", map[string]string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("input value for schema generation must be a struct, got string"))
			})

			It("should return an error if the value is a pointer to a non-struct", func() {
				invalidInput := "not a struct"
				_, err := prompterizer.MarshalResponseSchema(&invalidInput, map[string]string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("input value for schema generation must be a struct, got string"))
			})

			It("should return an error if there is an attempt to marshal without all the required template variables", func() {
				_, err := prompterizer.MarshalResponseSchema(TestPrompt{}, map[string]string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("missing variables in description: seriesName"))
			})

			It("should return an error when there are an incorrect number of params", func() {
				_, err := prompterizer.MarshalResponseSchema(WrongNumberOfParams{}, map[string]string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to parse field params for Field: missing either prompt property name or type"))
			})

			It("should return an error for an invalid field type", func() {
				_, err := prompterizer.MarshalResponseSchema(InvalidType{}, map[string]string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("unsupported field type"))
			})

			It("should return an error for a type mismatch", func() {
				_, err := prompterizer.MarshalResponseSchema(TypeMismatch{}, map[string]string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("type mismatch for field 'mismatchedField': Go type implies STRING, but prompt tag specifies INTEGER"))
			})

			It("should return an error for an unsupported type", func() {
				_, err := prompterizer.MarshalResponseSchema(UnsupportedType{}, map[string]string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("error marshaling property unsupportedField (Go field Field, type complex64): unsupported type kind for schema generation: complex64 (Go type: complex64)"))
			})
		})
	})
})
