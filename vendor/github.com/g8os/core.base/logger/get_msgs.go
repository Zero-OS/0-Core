package logger

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/g8os/core.base/pm"
	"github.com/g8os/core.base/pm/core"
	"github.com/g8os/core.base/pm/process"
	"github.com/g8os/core.base/utils"
)

type logQuery struct {
	JobID  string      `json:"jobid"`
	Levels interface{} `json:"levels"`
	Limit  int         `json:"limit"`
}

type getMsgsFunc struct {
	db *bolt.DB
}

const (
	cmdGetMsgs             = "get_msgs"
	cmdGetMsgsDefaultLimit = 1000
)

func getLevels(levels interface{}) ([]int, error) {
	var results []int

	if levels == nil {
		levels = "*"
	}

	//loading levels.
	if levels != nil {
		switch ls := levels.(type) {
		case string:
			var err error
			results, err = utils.Expand(ls)
			if err != nil {
				return nil, err
			}
		case []int:
			results = ls
		case []float64:
			//happens when unmarshaling from json
			results = make([]int, len(ls))
			for i := 0; i < len(ls); i++ {
				results[i] = int(ls[i])
			}
		}
	} else {
		levels = make([]int, 0)
	}

	return results, nil
}

func (fnc *getMsgsFunc) getMsgs(cmd *core.Command) (interface{}, error) {
	query := logQuery{}

	err := json.Unmarshal(*cmd.Arguments, &query)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse get_msgs query: %s", err)
	}

	if query.JobID == "" {
		return nil, fmt.Errorf("jobid is required")
	}

	levels, err := getLevels(query.Levels)
	if err != nil {
		return nil, err
	}

	var limit int
	if query.Limit != 0 {
		limit = query.Limit
	}

	if limit > cmdGetMsgsDefaultLimit {
		limit = cmdGetMsgsDefaultLimit
	}

	//we still can continue the query even if we have unmarshal errors.
	records := make([]map[string]interface{}, 0, cmdGetMsgsDefaultLimit)

	err = fnc.db.View(func(tx *bolt.Tx) error {
		logs := tx.Bucket([]byte("logs"))
		if logs == nil {
			return errors.New("Logs database is not initialized")
		}

		job := logs.Bucket([]byte(query.JobID))
		if job == nil {
			log.Errorf("Failed to open job bucket")
			return nil
		}
		cursor := job.Cursor()
		for key, value := cursor.Last(); key != nil && len(records) < limit; key, value = cursor.Prev() {
			row := make(map[string]interface{})
			err := json.Unmarshal(value, &row)
			if err != nil {
				log.Errorf("Failed to load job log '%s'", value)
				return err
			}
			if utils.In(levels, int(row["level"].(float64))) {
				records = append(records, row)
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return records, nil
}

func registerGetMsgsFunction(db *bolt.DB) {
	fnc := &getMsgsFunc{
		db: db,
	}

	pm.CmdMap[cmdGetMsgs] = process.NewInternalProcessFactory(fnc.getMsgs)
}
