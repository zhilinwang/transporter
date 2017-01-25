// Copyright 2012-present Oliver Eilhard. All rights reserved.
// Use of this source code is governed by a MIT-license.
// See http://olivere.mit-license.org/license.txt for details.

package elastic

// SumAggregation is a single-value metrics aggregation that sums up
// numeric values that are extracted from the aggregated documents.
// These values can be extracted either from specific numeric fields
// in the documents, or be generated by a provided script.
// See: http://www.elasticsearch.org/guide/en/elasticsearch/reference/current/search-aggregations-metrics-sum-aggregation.html
type SumAggregation struct {
	field           string
	script          *Script
	format          string
	subAggregations map[string]Aggregation
	meta            map[string]interface{}
}

func NewSumAggregation() *SumAggregation {
	return &SumAggregation{
		subAggregations: make(map[string]Aggregation),
	}
}

func (a *SumAggregation) Field(field string) *SumAggregation {
	a.field = field
	return a
}

func (a *SumAggregation) Script(script *Script) *SumAggregation {
	a.script = script
	return a
}

func (a *SumAggregation) Format(format string) *SumAggregation {
	a.format = format
	return a
}

func (a *SumAggregation) SubAggregation(name string, subAggregation Aggregation) *SumAggregation {
	a.subAggregations[name] = subAggregation
	return a
}

// Meta sets the meta data to be included in the aggregation response.
func (a *SumAggregation) Meta(metaData map[string]interface{}) *SumAggregation {
	a.meta = metaData
	return a
}

func (a *SumAggregation) Source() (interface{}, error) {
	// Example:
	//	{
	//    "aggs" : {
	//      "intraday_return" : { "sum" : { "field" : "change" } }
	//    }
	//	}
	// This method returns only the { "sum" : { "field" : "change" } } part.

	source := make(map[string]interface{})
	opts := make(map[string]interface{})
	source["sum"] = opts

	// ValuesSourceAggregationBuilder
	if a.field != "" {
		opts["field"] = a.field
	}
	if a.script != nil {
		src, err := a.script.Source()
		if err != nil {
			return nil, err
		}
		opts["script"] = src
	}
	if a.format != "" {
		opts["format"] = a.format
	}

	// AggregationBuilder (SubAggregations)
	if len(a.subAggregations) > 0 {
		aggsMap := make(map[string]interface{})
		source["aggregations"] = aggsMap
		for name, aggregate := range a.subAggregations {
			src, err := aggregate.Source()
			if err != nil {
				return nil, err
			}
			aggsMap[name] = src
		}
	}

	// Add Meta data if available
	if len(a.meta) > 0 {
		source["meta"] = a.meta
	}

	return source, nil
}
