package criteria

import (
	"github.com/viant/assertly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
)

//Criteria represents logical criteria
type Criteria struct {
	LogicalOperator string
	Criteria        []*Criterion
}

//IsTrue evaluates criteria with supplied context and state map . Dolar prefixed $expression will be expanded before evaluation.
func (c *Criteria) IsTrue(state data.Map) (bool, error) {
	if c.LogicalOperator == "||" {
		for _, criterion := range c.Criteria {
			result, err := criterion.IsTrue(state)
			if result || err != nil {
				return result, err
			}
		}
		return false, nil
	}
	for _, criterion := range c.Criteria {
		result, err := criterion.IsTrue(state)
		if !result || err != nil {
			return result, err
		}
	}
	return true, nil
}

//NewCriteria creates a new criteria for supplied logical operator and criteria
func NewCriteria(operator string, criteria ...*Criterion) *Criteria {
	return &Criteria{
		LogicalOperator: operator,
		Criteria:        criteria,
	}
}

//Criterion represent evaluation criterion
type Criterion struct {
	*Criteria
	LeftOperand  interface{}
	Operator     string
	RightOperand interface{}
}

func (c *Criterion) expandOperand(opperand interface{}, state data.Map) interface{} {
	if opperand == nil {
		return nil
	}
	return state.Expand(opperand)
}

//IsTrue evaluates criterion with supplied context and state map . Dolar prefixed $expression will be expanded before evaluation.
func (c *Criterion) IsTrue(state data.Map) (bool, error) {
	if c.Criteria != nil {
		return c.Criteria.IsTrue(state)
	}
	leftOperand := c.expandOperand(c.LeftOperand, state)
	rightOperand := c.expandOperand(c.RightOperand, state)

	var err error
	var leftNumber, rightNumber float64

	var rootPath = assertly.NewDataPath("/")
	var context = assertly.NewDefaultContext()

	switch c.Operator {
	case "=", ":":

		validation, err := assertly.AssertWithContext(rightOperand, leftOperand, rootPath, context)
		if err != nil {
			return false, err
		}
		return validation.FailedCount == 0, nil
	case "!=", "":
		if _, ok := leftOperand.(string); ok && rightOperand == nil {
			rightOperand = ""
		}
		validation, err := assertly.AssertWithContext(leftOperand, rightOperand, rootPath, context)
		if err != nil {
			return false, err
		}
		return validation.FailedCount > 0, nil
	case ">=":
		if leftNumber, err = toolbox.ToFloat(leftOperand); err == nil {
			if rightNumber, err = toolbox.ToFloat(rightOperand); err == nil {
				return leftNumber >= rightNumber, nil
			}
		}
	case "<=":
		if leftNumber, err = toolbox.ToFloat(leftOperand); err == nil {
			if rightNumber, err = toolbox.ToFloat(rightOperand); err == nil {
				return leftNumber <= rightNumber, nil
			}
		}

	case ">":
		if leftNumber, err = toolbox.ToFloat(leftOperand); err == nil {
			if rightNumber, err = toolbox.ToFloat(rightOperand); err == nil {
				return leftNumber > rightNumber, nil
			}
		}
	case "<":
		if leftNumber, err = toolbox.ToFloat(leftOperand); err == nil {
			if rightNumber, err = toolbox.ToFloat(rightOperand); err == nil {
				return leftNumber < rightNumber, nil
			}
		}
	}
	return false, err
}

//NewCriterion creates a new criterion
func NewCriterion(leftOperand interface{}, operator string, rightOperand interface{}) *Criterion {
	return &Criterion{
		LeftOperand:  leftOperand,
		Operator:     operator,
		RightOperand: rightOperand,
	}
}