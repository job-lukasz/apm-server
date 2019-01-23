// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package package_tests

import (
	"encoding/json"
	"testing"

	"github.com/elastic/apm-server/model/error/generated/schema"
	"github.com/elastic/apm-server/processor/stream"
	"github.com/elastic/apm-server/tests"
)

func errorProcSetup() *tests.ProcessorSetup {
	return &tests.ProcessorSetup{
		Proc:            &intakeTestProcessor{Processor: stream.Processor{MaxEventSize: lrSize}},
		FullPayloadPath: "../testdata/intake-v2/errors.ndjson",
		TemplatePaths: []string{
			"../../../model/error/_meta/fields.yml",
			"../../../_meta/fields.common.yml",
		},
		Schema:       schema.ModelSchema,
		SchemaPrefix: "error",
	}
}

func errorPayloadAttrsNotInFields() *tests.Set {
	return tests.NewSet(
		tests.Group("error.exception.attributes"),
		tests.Group("error.exception.stacktrace"),
		tests.Group("error.log.stacktrace"),
		tests.Group("context.user"),
	)
}

func errorFieldsNotInPayloadAttrs() *tests.Set {
	return tests.NewSet(
		"view errors", "error id icon",
		"context.http", "context.http.status_code",
		"host.ip",
		tests.Group("observer"),
		tests.Group("user"),
		tests.Group("client"),
	)
}

func errorPayloadAttrsNotInJsonSchema() *tests.Set {
	return tests.NewSet(
		"error",
		"error.log.stacktrace.vars.key",
		"error.exception.stacktrace.vars.key",
		"error.exception.attributes.foo",
		"error.context.request.headers.some-other-header",
		"error.context.request.headers.array",
		tests.Group("error.context.custom"),
		tests.Group("error.context.request.env"),
		tests.Group("error.context.request.cookies"),
		tests.Group("error.context.tags"),
	)
}

func errorRequiredKeys() *tests.Set {
	return tests.NewSet(
		"error",
		"error.id",
		"error.log",
		"error.exception",
		"error.exception.type",
		"error.exception.message",
		"error.log.message",
		"error.exception.stacktrace.filename",
		"error.exception.stacktrace.lineno",
		"error.log.stacktrace.filename",
		"error.log.stacktrace.lineno",
		"error.context.request.method",
		"error.context.request.url",

		"error.trace_id",
		"error.transaction_id",
		"error.parent_id",
	)
}

type val = []interface{}
type obj = map[string]interface{}

func errorCondRequiredKeys() map[string]tests.Condition {
	return map[string]tests.Condition{
		"error.exception":         tests.Condition{Absence: []string{"error.log"}},
		"error.exception.message": tests.Condition{Absence: []string{"error.exception.type"}},
		"error.exception.type":    tests.Condition{Absence: []string{"error.exception.message"}},
		"error.log":               tests.Condition{Absence: []string{"error.exception"}},

		"error.trace_id":       tests.Condition{Existence: obj{"error.parent_id": "abc123", "error.transaction_id": "abc123"}},
		"error.transaction_id": tests.Condition{Existence: obj{"error.parent_id": "abc123", "error.trace_id": "abc123"}},
		"error.parent_id":      tests.Condition{Existence: obj{"error.transaction_id": "abc123", "error.trace_id": "abc123"}},
	}
}

func errorKeywordExceptionKeys() *tests.Set {
	return tests.NewSet(
		"processor.event", "processor.name", "error.grouping_key",
		"context.tags",
		"view errors", "error id icon",

		// metadata fields
		tests.Group("agent"),
		tests.Group("container"),
		tests.Group("host"),
		tests.Group("kubernetes"),
		tests.Group("observer"),
		tests.Group("process"),
		tests.Group("service"),
		tests.Group("user"),
	)
}

func TestErrorPayloadAttrsMatchFields(t *testing.T) {
	errorProcSetup().PayloadAttrsMatchFields(t,
		errorPayloadAttrsNotInFields(),
		errorFieldsNotInPayloadAttrs())
}

func TestErrorPayloadAttrsMatchJsonSchema(t *testing.T) {
	errorProcSetup().PayloadAttrsMatchJsonSchema(t,
		errorPayloadAttrsNotInJsonSchema(),
		tests.NewSet("error.context.user.email"))
}

func TestErrorAttrsPresenceInError(t *testing.T) {
	errorProcSetup().AttrsPresence(t, errorRequiredKeys(), errorCondRequiredKeys())
}

func TestErrorKeywordLimitationOnErrorAttributes(t *testing.T) {
	errorProcSetup().KeywordLimitation(
		t,
		errorKeywordExceptionKeys(),
		[]tests.FieldTemplateMapping{
			{Template: "error."},
			{Template: "transaction.id", Mapping: "transaction_id"},
			{Template: "parent.id", Mapping: "parent_id"},
			{Template: "trace.id", Mapping: "trace_id"},
		},
	)
}

func TestPayloadDataForError(t *testing.T) {
	//// add test data for testing
	//// * specific edge cases
	//// * multiple allowed dataypes
	//// * regex pattern, time formats
	//// * length restrictions, other than keyword length restrictions
	errorProcSetup().DataValidation(t,
		[]tests.SchemaTestData{
			{Key: "error",
				Invalid: []tests.Invalid{{Msg: `/type`, Values: val{false}}}},
			{Key: "error.exception.code", Valid: val{"success", ""},
				Invalid: []tests.Invalid{{Msg: `exception/properties/code/type`, Values: val{false}}}},
			{Key: "error.exception.attributes", Valid: val{map[string]interface{}{}},
				Invalid: []tests.Invalid{{Msg: `exception/properties/attributes/type`, Values: val{123}}}},
			{Key: "error.timestamp",
				Valid: val{json.Number("1496170422281000")},
				Invalid: []tests.Invalid{
					{Msg: `timestamp/type`, Values: val{"1496170422281000"}}}},
			{Key: "error.log.stacktrace.post_context",
				Valid: val{[]interface{}{}, []interface{}{"context"}},
				Invalid: []tests.Invalid{
					{Msg: `log/properties/stacktrace/items/properties/post_context/items/type`, Values: val{[]interface{}{123}}},
					{Msg: `log/properties/stacktrace/items/properties/post_context/type`, Values: val{"test"}}}},
			{Key: "error.log.stacktrace.pre_context",
				Valid: val{[]interface{}{}, []interface{}{"context"}},
				Invalid: []tests.Invalid{
					{Msg: `log/properties/stacktrace/items/properties/pre_context/items/type`, Values: val{[]interface{}{123}}},
					{Msg: `log/properties/stacktrace/items/properties/pre_context/type`, Values: val{"test"}}}},
			{Key: "error.exception.stacktrace.post_context",
				Valid: val{[]interface{}{}, []interface{}{"context"}},
				Invalid: []tests.Invalid{
					{Msg: `exception/properties/stacktrace/items/properties/post_context/items/type`, Values: val{[]interface{}{123}}},
					{Msg: `exception/properties/stacktrace/items/properties/post_context/type`, Values: val{"test"}}}},
			{Key: "error.exception.stacktrace.pre_context",
				Valid: val{[]interface{}{}, []interface{}{"context"}},
				Invalid: []tests.Invalid{
					{Msg: `exception/properties/stacktrace/items/properties/pre_context/items/type`, Values: val{[]interface{}{123}}},
					{Msg: `exception/properties/stacktrace/items/properties/pre_context/type`, Values: val{"test"}}}},
			{Key: "error.context.custom",
				Valid: val{obj{"whatever": obj{"comes": obj{"end": -45}}}, obj{"whatever": 123}},
				Invalid: []tests.Invalid{
					{Msg: `context/properties/custom/additionalproperties`, Values: val{
						obj{"what.ever": 123}, obj{"what*ever": 123}, obj{"what\"ever": 123}}},
					{Msg: `context/properties/custom/type`, Values: val{"context"}}}},
			{Key: "error.context.request.body", Valid: val{tests.Str1025, obj{}},
				Invalid: []tests.Invalid{{Msg: `/context/properties/request/properties/body/type`, Values: val{102}}}},
			{Key: "error.context.request.env", Valid: val{obj{}},
				Invalid: []tests.Invalid{{Msg: `/context/properties/request/properties/env/type`, Values: val{102, "a"}}}},
			{Key: "error.context.request.cookies", Valid: val{obj{}},
				Invalid: []tests.Invalid{{Msg: `/context/properties/request/properties/cookies/type`, Values: val{102, "a"}}}},
			{Key: "error.context.tags",
				Valid: val{obj{tests.Str1024Special: tests.Str1024Special}, obj{tests.Str1024: 123.45}, obj{tests.Str1024: true}},
				Invalid: []tests.Invalid{
					{Msg: `context/properties/tags/type`, Values: val{"tags"}},
					{Msg: `context/properties/tags/patternproperties`, Values: val{obj{"invalid": tests.Str1025}, obj{tests.Str1024: obj{}}}},
					{Msg: `context/properties/tags/additionalproperties`, Values: val{obj{"invali*d": "hello"}, obj{"invali\"d": "hello"}, obj{"invali.d": "hello"}}}}},
			{Key: "error.context.user.id", Valid: val{123, tests.Str1024Special},
				Invalid: []tests.Invalid{
					{Msg: `context/properties/user/properties/id/type`, Values: val{obj{}}},
					{Msg: `context/properties/user/properties/id/maxlength`, Values: val{tests.Str1025}}}},
		})
}
