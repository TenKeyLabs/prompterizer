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

type TestPrompt struct {
	Ignored      string          `json:"ignored"`
	DocumentDate time.Time       `json:"documentDate" prompt:"documentDate,string,required"`
	PublishDate  time.Time       `json:"publishDate" prompt:"publishDate,string" prompt_description:"The date the document was published"`
	Title        string          `json:"title" prompt:"title,string" prompt_description:"The title of the document under the series {seriesName}"`
	FirstName    string          `json:"firstName" prompt:"firstName,string" prompt_aliases:"givenName"`
	LastName     string          `json:"lastName" prompt:"lastName,string" prompt_aliases:"surName,familyName"`
	Witness      string          `json:"witness" prompt:"witness,string" prompt_description:"The person from {seriesName} who witnessed the document signing." prompt_aliases:"witnessName,witnessedBy"`
	IsParsed     bool            `json:"isParsed" prompt:"isParsed,bool"`
	Amount       decimal.Decimal `json:"amount" prompt:"totalAmount,number"`
	Metadata     Metadata        `json:"metadata" prompt:"metadata,object"`
	Tags         []string        `json:"tags" prompt:"tags,string"`
	Events       []Event         `json:"events" prompt:"events,object"`
}

type InvalidType struct {
	Field string `json:"invalidField" prompt:"invalidField,"`
}
type WrongNumberOfParams struct {
	Field string `json:"wrongNumberOfParams" prompt:"wrongNumberOfParams"`
}

var _ = Describe("Ai Utils", func() {
	Describe("MarshalResponseSchema", func() {
		var schema *genai.Schema

		BeforeEach(func() {
			var err error
			schema, err = prompterizer.MarshalResponseSchema(TestPrompt{}, map[string]string{"seriesName": "Business 101"})
			Expect(err).ToNot(HaveOccurred())
		})

		Context("success", func() {
			It("should marshal a time.Time property", func() {
				Expect(schema).To(PointTo(MatchFields(IgnoreExtras, Fields{
					"Properties": HaveKey("documentDate"),
					"Required":   Equal([]string{"documentDate"}),
				})))

				Expect(schema.Properties["documentDate"]).To(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type": Equal(genai.TypeString),
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

			It("should marshal a decimal.Decimal property", func() {
				Expect(schema.Properties).To(HaveKey("totalAmount"))
				Expect(schema.Properties["totalAmount"]).To(PointTo(MatchFields(IgnoreExtras, Fields{
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

			It("should marshal an array of strings", func() {
				Expect(schema.Properties).To(HaveKey("tags"))
				Expect(schema.Properties["tags"]).To(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type": Equal(genai.TypeArray),
					"Items": PointTo(MatchFields(IgnoreExtras, Fields{
						"Type": Equal(genai.TypeString),
					})),
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

			It("should ignore fields without a prompt tag", func() {
				Expect(schema.Properties).ToNot(HaveKey("ignored"))
			})

			It("should render variables into the description", func() {
				Expect(schema.Properties).To(HaveKey("title"))
				Expect(schema.Properties["title"]).To(PointTo(MatchFields(IgnoreExtras, Fields{
					"Description": Equal("The title of the document under the series Business 101"),
				})))
			})
		})

		Context("errors", func() {
			It("should return an error when there are an incorrect number of params", func() {
				_, err := prompterizer.MarshalResponseSchema(WrongNumberOfParams{}, map[string]string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to parse field params for Field: missing either prompt property name or type"))
			})

			It("should return an error for an invalid field type", func() {
				_, err := prompterizer.MarshalResponseSchema(InvalidType{}, map[string]string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("unsupported property type"))
			})

			It("should return an error if there is an attempt to marshal without all the required template variables", func() {
				_, err := prompterizer.MarshalResponseSchema(TestPrompt{}, map[string]string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("missing variables in description: seriesName"))
			})
		})
	})
})
