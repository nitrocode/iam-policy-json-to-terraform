package converter

import (
	"strconv"
	"strings"
)

func escapeString(s string) string {
	// Escape TF special characters
	return strings.Replace(s, "$", "$$", -1)
}

func convertConditions(conditions map[string]map[string]stringOrStringArray) []hclCondition {
	result := make([]hclCondition, 0)
	for k, v := range conditions {
		for k2, v2 := range v {
			result = append(result, hclCondition{
				Test:     k,
				Variable: k2,
				Values:   convertStringOrStringArray(v2),
			})
		}
	}
	return result
}

func convertPrincipals(v stringOrMapWithStringOrStringArray) []hclPrincipal {
	if v == nil {
		return nil
	}

	stringPrincipal, stringOk := v.(string)
	if stringOk {
		return []hclPrincipal{
			{
				Type:        "*",
				Identifiers: []string{stringPrincipal},
			},
		}
	}
	return convertMapPrincipals(v.(map[string]interface{}))
}

func convertMapPrincipals(principals map[string]interface{}) []hclPrincipal {
	result := make([]hclPrincipal, 0)
	for k, v := range principals {
		result = append(result, hclPrincipal{
			Type:        k,
			Identifiers: convertStringOrStringArray(v),
		})
	}
	return result
}

func convertStringOrStringArray(v stringOrStringArray) []string {
	var resources []string

	switch v.(type) {
	case []interface{}:
		resourceArray, _ := v.([]interface{})
		resources = make([]string, len(resourceArray))
		for i, r := range resourceArray {
			resources[i] = escapeString(r.(string))
		}
	case string:
		resourceString, _ := v.(string)
		resources = []string{resourceString}
	case bool:
		resources = []string{strconv.FormatBool(v.(bool))}
	}

	return resources
}

func convertStatements(json jsonStatement) hclStatement {
	return hclStatement{
		Effect:        json.Effect,
		Sid:           json.Sid,
		Resources:     convertStringOrStringArray(json.Resource),
		NotResources:  convertStringOrStringArray(json.NotResource),
		Actions:       convertStringOrStringArray(json.Action),
		NotActions:    convertStringOrStringArray(json.NotAction),
		Conditions:    convertConditions(json.Condition),
		Principals:    convertPrincipals(json.Principal),
		NotPrincipals: convertPrincipals(json.NotPrincipal),
	}
}

// Convert reads a JSON policy document and return a string with a terraform policy document definition
func Convert(policyName string, b []byte) (string, error) {
	statementsFromJSON, err := decode(b)
	hclStatements := make([]hclStatement, len(statementsFromJSON))

	for i, s := range statementsFromJSON {
		hclStatements[i] = convertStatements(s)
	}

	if err != nil {
		return "", err
	}

	dataSource := hclDataSource{
		Type:       "aws_iam_policy_document",
		Name:       policyName,
		Statements: hclStatements,
	}

	tfFromStatements, err := encode(dataSource)

	if err != nil {
		return "", err
	}

	return tfFromStatements, err

}
