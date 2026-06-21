package main

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// specWithIntegerInvalidExample has type:integer with a string example.
// After Augment, the example should be removed.
var specWithIntegerInvalidExample = []byte(`{
  "openapi": "3.1.0",
  "info": {
    "title": "UniFi Protect",
    "version": "0.0.0"
  },
  "paths": {
    "/v1/items": {
      "get": {
        "summary": "List items",
        "operationId": "listItems",
        "responses": {
          "200": { "description": "OK" }
        }
      }
    }
  },
  "components": {
    "schemas": {
      "Device": {
        "type": "object",
        "properties": {
          "port": {
            "type": "integer",
            "example": "786435|720973"
          },
          "count": {
            "type": "integer",
            "example": 42
          }
        }
      }
    }
  }
}`)

// specWithACLRuleSchema has schema names containing spaces, simulating the
// upstream Network spec quirk.  A $ref pointing to the unclean name should
// be rewritten to the sanitized name.
var specWithACLRuleSchema = []byte(`{
  "openapi": "3.1.0",
  "info": {
    "title": "UniFi Network",
    "version": "0.0.0"
  },
  "paths": {
    "/v1/rules": {
      "get": {
        "summary": "List ACL rules",
        "operationId": "listACLRules",
        "responses": {
          "200": {
            "description": "OK",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/ACL rule"
                }
              }
            }
          }
        }
      }
    }
  },
  "components": {
    "schemas": {
      "ACL rule": {
        "type": "object",
        "properties": {
          "action": { "type": "string" }
        }
      }
    }
  }
}`)

// specWithCollisionSchemas has two schema names that both sanitize to the same
// identifier ("ACL_rule"): "ACL rule" (space→underscore) and "ACL_rule"
// (already clean). Augment must return an error rather than silently dropping
// one definition.
var specWithCollisionSchemas = []byte(`{
  "openapi": "3.1.0",
  "info": {
    "title": "UniFi Network",
    "version": "0.0.0"
  },
  "paths": {
    "/v1/items": {
      "get": {
        "summary": "List items",
        "operationId": "listItems",
        "responses": {
          "200": { "description": "OK" }
        }
      }
    }
  },
  "components": {
    "schemas": {
      "ACL rule": {
        "type": "object"
      },
      "ACL_rule": {
        "type": "object"
      }
    }
  }
}`)

// specWithNoVersionField has an info object that lacks a "version" key entirely.
var specWithNoVersionField = []byte(`{
  "openapi": "3.1.0",
  "info": {
    "title": "UniFi Protect"
  },
  "paths": {
    "/v1/items": {
      "get": {
        "summary": "List items",
        "operationId": "listItems",
        "responses": {
          "200": { "description": "OK" }
        }
      }
    }
  },
  "components": {}
}`)

// minimalProtectSpec is a small Protect-like OpenAPI 3.1.0 spec for testing Augment.
var minimalProtectSpec = []byte(`{
  "openapi": "3.1.0",
  "info": {
    "title": "UniFi Protect",
    "version": "0.0.0"
  },
  "paths": {
    "/v1/cameras/{id}": {
      "get": {
        "summary": "Get camera",
        "operationId": "getCamera",
        "responses": {
          "200": { "description": "OK" }
        }
      }
    },
    "/v1/viewers": {
      "get": {
        "summary": "List viewers",
        "operationId": "listViewers",
        "responses": {
          "200": { "description": "OK" }
        }
      }
    }
  },
  "components": {}
}`)

// minimalNetworkSpecForAugment is a small Network-like OpenAPI 3.1.0 spec for
// testing Augment app-specific overlay (separate from the Validate test copy).
var minimalNetworkSpecForAugment = []byte(`{
  "openapi": "3.1.0",
  "info": {
    "title": "UniFi Network",
    "version": "0.0.0"
  },
  "paths": {
    "/v1/devices": {
      "get": {
        "summary": "List devices",
        "operationId": "listDevices",
        "responses": {
          "200": { "description": "OK" }
        }
      }
    }
  },
  "components": {}
}`)

// specWithMissingResponseDescription contains a response object that has no
// "description" field — ensureResponseDescriptions should add one.
var specWithMissingResponseDescription = []byte(`{
  "openapi": "3.1.0",
  "info": {
    "title": "UniFi Protect",
    "version": "0.0.0"
  },
  "paths": {
    "/v1/items": {
      "get": {
        "summary": "List items",
        "operationId": "listItems",
        "responses": {
          "200": {}
        }
      }
    }
  },
  "components": {}
}`)

// specWithInvalidEnumExample has an enum field whose "example" is not a member.
// After Augment, the example should be removed.
var specWithInvalidEnumExample = []byte(`{
  "openapi": "3.1.0",
  "info": {
    "title": "UniFi Protect",
    "version": "0.0.0"
  },
  "paths": {
    "/v1/rules": {
      "get": {
        "summary": "List rules",
        "operationId": "listRules",
        "responses": {
          "200": { "description": "OK" }
        }
      }
    }
  },
  "components": {
    "schemas": {
      "Rule": {
        "type": "object",
        "properties": {
          "action": {
            "type": "string",
            "enum": ["ALLOW", "BLOCK"],
            "example": "ALLOW|BLOCK"
          },
          "mode": {
            "type": "string",
            "enum": ["ON", "OFF"],
            "example": "ON"
          }
        }
      }
    }
  }
}`)

var _ = Describe("Augment", func() {
	var result []byte

	BeforeEach(func() {
		var err error
		result, err = Augment("protect", "v7.1.46", minimalProtectSpec)
		Expect(err).NotTo(HaveOccurred())
		Expect(result).NotTo(BeEmpty())
	})

	It("injects ApiKeyAuth security scheme", func() {
		var doc map[string]any
		Expect(json.Unmarshal(result, &doc)).To(Succeed())

		components, ok := doc["components"].(map[string]any)
		Expect(ok).To(BeTrue(), "components should be a map")

		schemes, ok := components["securitySchemes"].(map[string]any)
		Expect(ok).To(BeTrue(), "securitySchemes should be a map")

		apiKey, ok := schemes["ApiKeyAuth"].(map[string]any)
		Expect(ok).To(BeTrue(), "ApiKeyAuth should be present")
		Expect(apiKey["name"]).To(Equal("X-API-KEY"))
		Expect(apiKey["in"]).To(Equal("header"))
	})

	It("injects global security as [{ApiKeyAuth: []}]", func() {
		var doc map[string]any
		Expect(json.Unmarshal(result, &doc)).To(Succeed())

		security, ok := doc["security"].([]any)
		Expect(ok).To(BeTrue(), "security should be a list")
		Expect(security).To(HaveLen(1), "should have exactly one security requirement")

		req, ok := security[0].(map[string]any)
		Expect(ok).To(BeTrue(), "security[0] should be a map")

		scopes, ok := req["ApiKeyAuth"].([]any)
		Expect(ok).To(BeTrue(), "ApiKeyAuth should be present in security requirement")
		Expect(scopes).To(BeEmpty(), "ApiKeyAuth scopes should be empty []")
	})

	It("injects two servers with the expected URLs", func() {
		var doc map[string]any
		Expect(json.Unmarshal(result, &doc)).To(Succeed())

		servers, ok := doc["servers"].([]any)
		Expect(ok).To(BeTrue(), "servers should be a list")
		Expect(servers).To(HaveLen(2), "should have exactly two servers")

		serverURLs := make([]string, 0, 2)
		for _, s := range servers {
			sm, ok := s.(map[string]any)
			Expect(ok).To(BeTrue())
			serverURLs = append(serverURLs, sm["url"].(string))
		}
		Expect(serverURLs).To(ContainElement("https://{host}/proxy/{app}/integration"))
		Expect(serverURLs).To(ContainElement("https://api.ui.com/v1/connector/consoles/{consoleId}/{app}/integration"))
	})

	It("local server has host and app variables", func() {
		var doc map[string]any
		Expect(json.Unmarshal(result, &doc)).To(Succeed())

		servers := doc["servers"].([]any)
		var localServer map[string]any
		for _, s := range servers {
			sm := s.(map[string]any)
			if sm["url"] == "https://{host}/proxy/{app}/integration" {
				localServer = sm
			}
		}
		Expect(localServer).NotTo(BeNil(), "local server should be present")

		vars, ok := localServer["variables"].(map[string]any)
		Expect(ok).To(BeTrue(), "local server should have variables")
		Expect(vars).To(HaveKey("host"))
		Expect(vars).To(HaveKey("app"))
	})

	It("per-app server app variable default is 'protect' when augmenting protect", func() {
		var doc map[string]any
		Expect(json.Unmarshal(result, &doc)).To(Succeed())

		servers := doc["servers"].([]any)
		for _, s := range servers {
			sm := s.(map[string]any)
			vars, ok := sm["variables"].(map[string]any)
			Expect(ok).To(BeTrue())
			appVar, ok := vars["app"].(map[string]any)
			Expect(ok).To(BeTrue(), "app variable should be a map in server %s", sm["url"])
			Expect(appVar["default"]).To(Equal("protect"), "app.default should be 'protect' in server %s", sm["url"])
		}
	})

	It("pins info.version without leading v", func() {
		var doc map[string]any
		Expect(json.Unmarshal(result, &doc)).To(Succeed())

		info, ok := doc["info"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(info["version"]).To(Equal("7.1.46"))
	})

	It("synthesizes tag for /v1/cameras/{id} GET", func() {
		var doc map[string]any
		Expect(json.Unmarshal(result, &doc)).To(Succeed())

		paths, ok := doc["paths"].(map[string]any)
		Expect(ok).To(BeTrue())

		cameraPath, ok := paths["/v1/cameras/{id}"].(map[string]any)
		Expect(ok).To(BeTrue())

		getOp, ok := cameraPath["get"].(map[string]any)
		Expect(ok).To(BeTrue())

		tags, ok := getOp["tags"].([]any)
		Expect(ok).To(BeTrue(), "get operation should have tags")
		Expect(tags).To(ContainElement("Cameras"))
	})

	It("is deterministic across two calls", func() {
		second, err := Augment("protect", "v7.1.46", minimalProtectSpec)
		Expect(err).NotTo(HaveOccurred())
		Expect(second).To(Equal(result))
	})

	Context("network app overlay", func() {
		var networkResult []byte

		BeforeEach(func() {
			var err error
			networkResult, err = Augment("network", "v10.3.58", minimalNetworkSpecForAugment)
			Expect(err).NotTo(HaveOccurred())
		})

		It("per-app server app variable default is 'network' when augmenting network", func() {
			var doc map[string]any
			Expect(json.Unmarshal(networkResult, &doc)).To(Succeed())

			servers := doc["servers"].([]any)
			for _, s := range servers {
				sm := s.(map[string]any)
				vars, ok := sm["variables"].(map[string]any)
				Expect(ok).To(BeTrue())
				appVar, ok := vars["app"].(map[string]any)
				Expect(ok).To(BeTrue())
				Expect(appVar["default"]).To(Equal("network"), "app.default should be 'network' in server %s", sm["url"])
			}
		})
	})

	Context("ensureResponseDescriptions", func() {
		It("adds description to a response object missing one", func() {
			out, err := Augment("protect", "v7.1.46", specWithMissingResponseDescription)
			Expect(err).NotTo(HaveOccurred())

			var doc map[string]any
			Expect(json.Unmarshal(out, &doc)).To(Succeed())

			paths := doc["paths"].(map[string]any)
			itemsPath := paths["/v1/items"].(map[string]any)
			getOp := itemsPath["get"].(map[string]any)
			responses := getOp["responses"].(map[string]any)
			resp200, ok := responses["200"].(map[string]any)
			Expect(ok).To(BeTrue())
			_, hasDesc := resp200["description"]
			Expect(hasDesc).To(BeTrue(), "response 200 should have a description after Augment")
		})
	})

	Context("stripInvalidEnumExamples", func() {
		It("removes example when it is not a member of enum", func() {
			out, err := Augment("protect", "v7.1.46", specWithInvalidEnumExample)
			Expect(err).NotTo(HaveOccurred())

			var doc map[string]any
			Expect(json.Unmarshal(out, &doc)).To(Succeed())

			components := doc["components"].(map[string]any)
			schemas := components["schemas"].(map[string]any)
			rule := schemas["Rule"].(map[string]any)
			props := rule["properties"].(map[string]any)

			// "ALLOW|BLOCK" is not in ["ALLOW","BLOCK"] → example should be removed.
			actionProp := props["action"].(map[string]any)
			Expect(actionProp).NotTo(HaveKey("example"), "invalid enum example 'ALLOW|BLOCK' should be stripped")
		})

		It("preserves example when it IS a valid enum member", func() {
			out, err := Augment("protect", "v7.1.46", specWithInvalidEnumExample)
			Expect(err).NotTo(HaveOccurred())

			var doc map[string]any
			Expect(json.Unmarshal(out, &doc)).To(Succeed())

			components := doc["components"].(map[string]any)
			schemas := components["schemas"].(map[string]any)
			rule := schemas["Rule"].(map[string]any)
			props := rule["properties"].(map[string]any)

			// "ON" IS in ["ON","OFF"] → example should be preserved.
			modeProp := props["mode"].(map[string]any)
			Expect(modeProp).To(HaveKey("example"), "valid enum example 'ON' should be preserved")
			Expect(modeProp["example"]).To(Equal("ON"))
		})
	})

	Context("stripInvalidTypeExamples", func() {
		It("removes a string example from a type:integer schema", func() {
			out, err := Augment("protect", "v7.1.46", specWithIntegerInvalidExample)
			Expect(err).NotTo(HaveOccurred())

			var doc map[string]any
			Expect(json.Unmarshal(out, &doc)).To(Succeed())

			components := doc["components"].(map[string]any)
			schemas := components["schemas"].(map[string]any)
			device := schemas["Device"].(map[string]any)
			props := device["properties"].(map[string]any)

			// "786435|720973" is a string on type:integer → should be stripped.
			portProp := props["port"].(map[string]any)
			Expect(portProp).NotTo(HaveKey("example"),
				"string example on integer field should be stripped")
		})

		It("preserves a numeric example that matches type:integer", func() {
			out, err := Augment("protect", "v7.1.46", specWithIntegerInvalidExample)
			Expect(err).NotTo(HaveOccurred())

			var doc map[string]any
			Expect(json.Unmarshal(out, &doc)).To(Succeed())

			components := doc["components"].(map[string]any)
			schemas := components["schemas"].(map[string]any)
			device := schemas["Device"].(map[string]any)
			props := device["properties"].(map[string]any)

			// 42 is a valid integer example → should be preserved.
			countProp := props["count"].(map[string]any)
			Expect(countProp).To(HaveKey("example"),
				"numeric example on integer field should be preserved")
		})
	})

	Context("sanitizeSchemaIdentifiers and rewriteRefs", func() {
		It("sanitizes 'ACL rule' schema key and rewrites matching $ref", func() {
			out, err := Augment("network", "v10.3.58", specWithACLRuleSchema)
			Expect(err).NotTo(HaveOccurred())

			var doc map[string]any
			Expect(json.Unmarshal(out, &doc)).To(Succeed())

			components := doc["components"].(map[string]any)
			schemas := components["schemas"].(map[string]any)

			// The original "ACL rule" key should be gone.
			Expect(schemas).NotTo(HaveKey("ACL rule"),
				"original schema key with space should have been removed")
			// The sanitized "ACL_rule" key should be present.
			Expect(schemas).To(HaveKey("ACL_rule"),
				"sanitized schema key 'ACL_rule' should exist")

			// The $ref in the response should also be rewritten.
			paths := doc["paths"].(map[string]any)
			rulesPath := paths["/v1/rules"].(map[string]any)
			getOp := rulesPath["get"].(map[string]any)
			responses := getOp["responses"].(map[string]any)
			resp200 := responses["200"].(map[string]any)
			content := resp200["content"].(map[string]any)
			jsonContent := content["application/json"].(map[string]any)
			schema := jsonContent["schema"].(map[string]any)
			Expect(schema["$ref"]).To(Equal("#/components/schemas/ACL_rule"),
				"$ref should be rewritten to the sanitized schema name")
		})

		It("returns an error when two schema names collide after sanitization", func() {
			_, err := Augment("network", "v10.3.58", specWithCollisionSchemas)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("collision"),
				"error message should mention 'collision'")
		})

		It("rewrites discriminator.mapping values that reference a renamed schema", func() {
			// specWithDiscriminatorMapping has a schema named "Derived entity metadata"
			// (contains spaces) that must be sanitized, and a discriminator.mapping
			// value that points to it via "#/components/schemas/Derived entity metadata".
			// After Augment, BOTH the schema key and the mapping value must be rewritten.
			specWithDiscriminatorMapping := []byte(`{
  "openapi": "3.1.0",
  "info": {
    "title": "UniFi Network",
    "version": "0.0.0"
  },
  "paths": {
    "/v1/items": {
      "get": {
        "summary": "List items",
        "operationId": "listItems",
        "responses": {
          "200": { "description": "OK" }
        }
      }
    }
  },
  "components": {
    "schemas": {
      "Derived entity metadata": {
        "type": "object",
        "properties": { "origin": { "type": "string" } }
      },
      "Metadata": {
        "discriminator": {
          "propertyName": "origin",
          "mapping": {
            "DERIVED": "#/components/schemas/Derived entity metadata"
          }
        },
        "oneOf": [
          { "$ref": "#/components/schemas/Derived entity metadata" }
        ]
      }
    }
  }
}`)

			out, err := Augment("network", "v10.3.58", specWithDiscriminatorMapping)
			Expect(err).NotTo(HaveOccurred())

			var doc map[string]any
			Expect(json.Unmarshal(out, &doc)).To(Succeed())

			components := doc["components"].(map[string]any)
			schemas := components["schemas"].(map[string]any)

			// The schema key must be sanitized.
			Expect(schemas).NotTo(HaveKey("Derived entity metadata"),
				"original schema key with spaces should have been removed")
			Expect(schemas).To(HaveKey("Derived_entity_metadata"),
				"sanitized schema key should exist")

			// The discriminator.mapping value must be rewritten.
			metadata := schemas["Metadata"].(map[string]any)
			discriminator := metadata["discriminator"].(map[string]any)
			mapping := discriminator["mapping"].(map[string]any)
			Expect(mapping["DERIVED"]).To(Equal("#/components/schemas/Derived_entity_metadata"),
				"discriminator.mapping value should be rewritten to use the sanitized schema name")

			// The $ref in the oneOf must also be rewritten (existing behaviour).
			oneOf := metadata["oneOf"].([]any)
			Expect(oneOf).To(HaveLen(1))
			firstRef := oneOf[0].(map[string]any)
			Expect(firstRef["$ref"]).To(Equal("#/components/schemas/Derived_entity_metadata"),
				"$ref in oneOf should also be rewritten to the sanitized schema name")
		})
	})

	Context("deduplicateAllOfDiscriminators", func() {
		// specWithMultiDiscriminatorAllOf simulates the upstream Network spec pattern
		// where a single allOf contains multiple items that each declare a
		// discriminator on the same propertyName "origin".  After Augment, only
		// the first item should retain its discriminator; the others should have
		// theirs stripped (but keep their other fields intact).
		var specWithMultiDiscriminatorAllOf = []byte(`{
  "openapi": "3.1.0",
  "info": {
    "title": "UniFi Network",
    "version": "0.0.0"
  },
  "paths": {
    "/v1/items": {
      "get": {
        "summary": "List items",
        "operationId": "listItems",
        "responses": {
          "200": { "description": "OK" }
        }
      }
    }
  },
  "components": {
    "schemas": {
      "Metadata": {
        "allOf": [
          {
            "discriminator": {
              "propertyName": "origin",
              "mapping": { "DERIVED": "#/components/schemas/DerivedMeta", "USER_DEFINED": "#/components/schemas/UserMeta" }
            },
            "properties": { "origin": { "type": "string" } },
            "required": ["origin"],
            "title": "Entity metadata"
          },
          {
            "discriminator": {
              "propertyName": "origin",
              "mapping": { "USER_DEFINED": "#/components/schemas/UserMeta" }
            },
            "properties": { "origin": { "type": "string" } },
            "required": ["origin"],
            "title": "User metadata"
          },
          {
            "discriminator": {
              "propertyName": "origin",
              "mapping": { "DERIVED": "#/components/schemas/DerivedMeta" }
            },
            "properties": { "origin": { "type": "string" } },
            "required": ["origin"],
            "title": "Derived metadata"
          }
        ]
      }
    }
  }
}`)

		It("keeps discriminator only on the first allOf item with a given propertyName", func() {
			out, err := Augment("network", "v10.3.58", specWithMultiDiscriminatorAllOf)
			Expect(err).NotTo(HaveOccurred())

			var doc map[string]any
			Expect(json.Unmarshal(out, &doc)).To(Succeed())

			components := doc["components"].(map[string]any)
			schemas := components["schemas"].(map[string]any)
			metadata := schemas["Metadata"].(map[string]any)
			allOf := metadata["allOf"].([]any)
			Expect(allOf).To(HaveLen(3))

			// First item retains its discriminator.
			first := allOf[0].(map[string]any)
			Expect(first).To(HaveKey("discriminator"),
				"first allOf item should retain its discriminator")

			// Second and third items should have had their discriminator stripped.
			second := allOf[1].(map[string]any)
			Expect(second).NotTo(HaveKey("discriminator"),
				"duplicate discriminator should be stripped from second allOf item")

			third := allOf[2].(map[string]any)
			Expect(third).NotTo(HaveKey("discriminator"),
				"duplicate discriminator should be stripped from third allOf item")
		})

		It("preserves non-discriminator fields on de-duplicated allOf items", func() {
			out, err := Augment("network", "v10.3.58", specWithMultiDiscriminatorAllOf)
			Expect(err).NotTo(HaveOccurred())

			var doc map[string]any
			Expect(json.Unmarshal(out, &doc)).To(Succeed())

			components := doc["components"].(map[string]any)
			schemas := components["schemas"].(map[string]any)
			metadata := schemas["Metadata"].(map[string]any)
			allOf := metadata["allOf"].([]any)

			// Second item lost its discriminator but should still have properties, required, title.
			second := allOf[1].(map[string]any)
			Expect(second).To(HaveKey("properties"),
				"properties should be preserved after discriminator is stripped")
			Expect(second).To(HaveKey("required"),
				"required should be preserved after discriminator is stripped")
			Expect(second).To(HaveKey("title"),
				"title should be preserved after discriminator is stripped")
			Expect(second["title"]).To(Equal("User metadata"))
		})
	})

	Context("deduplicateResponseSchemas", func() {
		// specWithInlineResponseSchemas has two paths whose response body schemas
		// are identical to (and inline copies of) a named component schema.
		// After Augment, both response body schemas should be replaced with $refs.
		var specWithInlineResponseSchemas = []byte(`{
  "openapi": "3.1.0",
  "info": {
    "title": "UniFi Protect",
    "version": "0.0.0"
  },
  "paths": {
    "/v1/cameras": {
      "get": {
        "summary": "List cameras",
        "operationId": "listCameras",
        "responses": {
          "200": {
            "description": "OK",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "properties": { "modelKey": { "type": "string" } }
                }
              }
            }
          }
        }
      }
    },
    "/v1/cameras/{id}": {
      "get": {
        "summary": "Get camera",
        "operationId": "getCamera",
        "responses": {
          "200": {
            "description": "OK",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "properties": { "modelKey": { "type": "string" } }
                }
              }
            }
          }
        }
      }
    }
  },
  "components": {
    "schemas": {
      "camera": {
        "type": "object",
        "properties": { "modelKey": { "type": "string" } }
      }
    }
  }
}`)

		It("replaces inline response schemas with $refs when they match a component schema", func() {
			out, err := Augment("protect", "v7.1.46", specWithInlineResponseSchemas)
			Expect(err).NotTo(HaveOccurred())

			var doc map[string]any
			Expect(json.Unmarshal(out, &doc)).To(Succeed())

			paths := doc["paths"].(map[string]any)

			cameraListPath := paths["/v1/cameras"].(map[string]any)
			listOp := cameraListPath["get"].(map[string]any)
			listResp := listOp["responses"].(map[string]any)["200"].(map[string]any)
			listSchema := listResp["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
			Expect(listSchema).To(HaveKey("$ref"),
				"inline response schema matching component should be replaced with $ref")
			Expect(listSchema["$ref"]).To(Equal("#/components/schemas/camera"))

			cameraGetPath := paths["/v1/cameras/{id}"].(map[string]any)
			getOp := cameraGetPath["get"].(map[string]any)
			getResp := getOp["responses"].(map[string]any)["200"].(map[string]any)
			getSchema := getResp["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
			Expect(getSchema).To(HaveKey("$ref"),
				"second inline response schema should also be replaced with $ref")
			Expect(getSchema["$ref"]).To(Equal("#/components/schemas/camera"))
		})
	})

	Context("collapseArrayScalarAnyOf", func() {
		// specWithArrayScalarAnyOfParam has a query parameter whose schema is
		// anyOf: [array-of-X, X], which trips oapi-codegen into generating
		// conflicting type declarations.
		var specWithArrayScalarAnyOfParam = []byte(`{
  "openapi": "3.1.0",
  "info": {
    "title": "UniFi Protect",
    "version": "0.0.0"
  },
  "paths": {
    "/v1/cameras/{id}/rtsps-stream": {
      "delete": {
        "summary": "Delete RTSPS stream",
        "operationId": "deleteCameraRtspsStream",
        "parameters": [
          {
            "name": "qualities",
            "in": "query",
            "required": true,
            "schema": {
              "anyOf": [
                {
                  "type": "array",
                  "items": { "enum": ["high","medium","low"], "type": "string" }
                },
                { "enum": ["high","medium","low"], "type": "string" }
              ],
              "description": "Quality levels to remove."
            }
          }
        ],
        "responses": {
          "204": { "description": "Deleted" }
        }
      }
    }
  },
  "components": {}
}`)

		It("collapses anyOf [array-of-X, X] to just array-of-X", func() {
			out, err := Augment("protect", "v7.1.46", specWithArrayScalarAnyOfParam)
			Expect(err).NotTo(HaveOccurred())

			var doc map[string]any
			Expect(json.Unmarshal(out, &doc)).To(Succeed())

			paths := doc["paths"].(map[string]any)
			cameraPath := paths["/v1/cameras/{id}/rtsps-stream"].(map[string]any)
			deleteOp := cameraPath["delete"].(map[string]any)
			params := deleteOp["parameters"].([]any)
			Expect(params).To(HaveLen(1))
			qualitiesParam := params[0].(map[string]any)
			schema := qualitiesParam["schema"].(map[string]any)

			Expect(schema).NotTo(HaveKey("anyOf"),
				"anyOf should be collapsed away")
			Expect(schema["type"]).To(Equal("array"),
				"collapsed schema should be the array form")
			Expect(schema).To(HaveKey("items"),
				"collapsed schema should retain items")
		})
	})

	Context("inlineDiscriminatedOneOfToRefs", func() {
		// specWithInlineDiscriminatedOneOf simulates the Protect spec pattern where
		// a discriminated oneOf contains inline objects instead of $refs.
		var specWithInlineDiscriminatedOneOf = []byte(`{
  "openapi": "3.1.0",
  "info": {
    "title": "UniFi Protect",
    "version": "0.0.0"
  },
  "paths": {
    "/v1/items": {
      "get": {
        "summary": "List items",
        "operationId": "listItems",
        "responses": {
          "200": { "description": "OK" }
        }
      }
    }
  },
  "components": {
    "schemas": {
      "camera": { "type": "object", "properties": { "modelKey": { "type": "string" } } },
      "nvr":    { "type": "object", "properties": { "modelKey": { "type": "string" } } },
      "device": {
        "discriminator": {
          "propertyName": "modelKey",
          "mapping": {
            "camera": "#/components/schemas/camera",
            "nvr":    "#/components/schemas/nvr"
          }
        },
        "oneOf": [
          { "title": "camera", "type": "object", "properties": { "modelKey": { "type": "string" } } },
          { "title": "nvr",    "type": "object", "properties": { "modelKey": { "type": "string" } } }
        ]
      }
    }
  }
}`)

		It("replaces inline oneOf items with $refs when title matches a component schema", func() {
			out, err := Augment("protect", "v7.1.46", specWithInlineDiscriminatedOneOf)
			Expect(err).NotTo(HaveOccurred())

			var doc map[string]any
			Expect(json.Unmarshal(out, &doc)).To(Succeed())

			components := doc["components"].(map[string]any)
			schemas := components["schemas"].(map[string]any)
			device := schemas["device"].(map[string]any)
			oneOf := device["oneOf"].([]any)
			Expect(oneOf).To(HaveLen(2))

			first := oneOf[0].(map[string]any)
			Expect(first).To(HaveKey("$ref"),
				"inline camera schema should be replaced with a $ref")
			Expect(first["$ref"]).To(Equal("#/components/schemas/camera"))

			second := oneOf[1].(map[string]any)
			Expect(second).To(HaveKey("$ref"),
				"inline nvr schema should be replaced with a $ref")
			Expect(second["$ref"]).To(Equal("#/components/schemas/nvr"))
		})
	})

	Context("collapseOneOfNullable", func() {
		// specWithOneOfNullable exercises the OpenAPI 3.1 pattern of
		// oneOf: [{...type:string}, {type:"null"}] that oapi-codegen cannot handle.
		var specWithOneOfNullable = []byte(`{
  "openapi": "3.1.0",
  "info": {
    "title": "UniFi Protect",
    "version": "0.0.0"
  },
  "paths": {
    "/v1/items": {
      "get": {
        "summary": "List items",
        "operationId": "listItems",
        "responses": {
          "200": { "description": "OK" }
        }
      }
    }
  },
  "components": {
    "schemas": {
      "Device": {
        "type": "object",
        "properties": {
          "name": {
            "oneOf": [
              { "type": "string", "description": "The device name.", "title": "name" },
              { "type": "null" }
            ]
          },
          "count": {
            "type": "integer"
          }
        }
      }
    }
  }
}`)

		It("collapses oneOf: [S, {type:'null'}] to S with nullable:true", func() {
			out, err := Augment("protect", "v7.1.46", specWithOneOfNullable)
			Expect(err).NotTo(HaveOccurred())

			var doc map[string]any
			Expect(json.Unmarshal(out, &doc)).To(Succeed())

			components := doc["components"].(map[string]any)
			schemas := components["schemas"].(map[string]any)
			device := schemas["Device"].(map[string]any)
			props := device["properties"].(map[string]any)

			nameProp := props["name"].(map[string]any)
			Expect(nameProp).NotTo(HaveKey("oneOf"),
				"oneOf should be collapsed away")
			Expect(nameProp["type"]).To(Equal("string"),
				"collapsed schema should have type from the non-null oneOf member")
			Expect(nameProp["nullable"]).To(Equal(true),
				"collapsed schema should have nullable:true")
			Expect(nameProp["description"]).To(Equal("The device name."),
				"description from the non-null oneOf member should be preserved")
		})

		It("leaves non-nullable schemas unchanged", func() {
			out, err := Augment("protect", "v7.1.46", specWithOneOfNullable)
			Expect(err).NotTo(HaveOccurred())

			var doc map[string]any
			Expect(json.Unmarshal(out, &doc)).To(Succeed())

			components := doc["components"].(map[string]any)
			schemas := components["schemas"].(map[string]any)
			device := schemas["Device"].(map[string]any)
			props := device["properties"].(map[string]any)

			countProp := props["count"].(map[string]any)
			Expect(countProp["type"]).To(Equal("integer"),
				"plain integer type should be left unchanged")
			Expect(countProp).NotTo(HaveKey("nullable"),
				"non-nullable schema should not gain a nullable key")
		})
	})

	Context("normalizeNullableTypeArrays", func() {
		// specWithNullableTypeArrays exercises the OpenAPI 3.1 pattern of
		// ["T","null"] type arrays that oapi-codegen cannot handle.
		var specWithNullableTypeArrays = []byte(`{
  "openapi": "3.1.0",
  "info": {
    "title": "UniFi Protect",
    "version": "0.0.0"
  },
  "paths": {
    "/v1/items": {
      "get": {
        "summary": "List items",
        "operationId": "listItems",
        "responses": {
          "200": { "description": "OK" }
        }
      }
    }
  },
  "components": {
    "schemas": {
      "Device": {
        "type": "object",
        "properties": {
          "name": { "type": ["string", "null"] },
          "count": { "type": ["null", "number"] },
          "enabled": { "type": ["boolean", "null"] },
          "tags": { "type": "string" }
        }
      }
    }
  }
}`)

		It("converts [T, null] type arrays to nullable scalar type", func() {
			out, err := Augment("protect", "v7.1.46", specWithNullableTypeArrays)
			Expect(err).NotTo(HaveOccurred())

			var doc map[string]any
			Expect(json.Unmarshal(out, &doc)).To(Succeed())

			components := doc["components"].(map[string]any)
			schemas := components["schemas"].(map[string]any)
			device := schemas["Device"].(map[string]any)
			props := device["properties"].(map[string]any)

			// ["string","null"] → type:"string", nullable:true
			nameProp := props["name"].(map[string]any)
			Expect(nameProp["type"]).To(Equal("string"),
				"[string,null] should be normalized to type:string")
			Expect(nameProp["nullable"]).To(Equal(true),
				"[string,null] should set nullable:true")

			// ["null","number"] → type:"number", nullable:true
			countProp := props["count"].(map[string]any)
			Expect(countProp["type"]).To(Equal("number"),
				"[null,number] should be normalized to type:number")
			Expect(countProp["nullable"]).To(Equal(true),
				"[null,number] should set nullable:true")

			// ["boolean","null"] → type:"boolean", nullable:true
			enabledProp := props["enabled"].(map[string]any)
			Expect(enabledProp["type"]).To(Equal("boolean"),
				"[boolean,null] should be normalized to type:boolean")
			Expect(enabledProp["nullable"]).To(Equal(true),
				"[boolean,null] should set nullable:true")
		})

		It("leaves scalar type unchanged", func() {
			out, err := Augment("protect", "v7.1.46", specWithNullableTypeArrays)
			Expect(err).NotTo(HaveOccurred())

			var doc map[string]any
			Expect(json.Unmarshal(out, &doc)).To(Succeed())

			components := doc["components"].(map[string]any)
			schemas := components["schemas"].(map[string]any)
			device := schemas["Device"].(map[string]any)
			props := device["properties"].(map[string]any)

			// "type": "string" should be left as-is with no nullable key.
			tagsProp := props["tags"].(map[string]any)
			Expect(tagsProp["type"]).To(Equal("string"),
				"scalar type should be unchanged")
			Expect(tagsProp).NotTo(HaveKey("nullable"),
				"scalar type should not have nullable set")
		})
	})

	Context("info.version pinning when version key is absent", func() {
		It("creates info.version when the key does not exist upstream", func() {
			out, err := Augment("protect", "v7.1.46", specWithNoVersionField)
			Expect(err).NotTo(HaveOccurred())

			var doc map[string]any
			Expect(json.Unmarshal(out, &doc)).To(Succeed())

			info, ok := doc["info"].(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(info["version"]).To(Equal("7.1.46"),
				"info.version should be pinned even when absent in the upstream spec")
		})
	})
})
