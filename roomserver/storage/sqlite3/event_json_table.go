// Copyright 2017-2018 New Vector Ltd
// Copyright 2019-2020 The Matrix.org Foundation C.I.C.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sqlite3

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/matrix-org/dendrite/roomserver/types"
)

const eventJSONSchema = `
  CREATE TABLE IF NOT EXISTS roomserver_event_json (
    event_nid INTEGER NOT NULL PRIMARY KEY,
    event_json TEXT NOT NULL
  );
`

const insertEventJSONSQL = `
	INSERT INTO roomserver_event_json (event_nid, event_json) VALUES ($1, $2)
	  ON CONFLICT DO NOTHING
`

// Bulk event JSON lookup by numeric event ID.
// Sort by the numeric event ID.
// This means that we can use binary search to lookup by numeric event ID.
const bulkSelectEventJSONSQL = `
	SELECT event_nid, event_json FROM roomserver_event_json
	  WHERE event_nid IN ($1)
	  ORDER BY event_nid ASC
`

type eventJSONStatements struct {
	insertEventJSONStmt     *sql.Stmt
	bulkSelectEventJSONStmt *sql.Stmt
}

func (s *eventJSONStatements) prepare(db *sql.DB) (err error) {
	_, err = db.Exec(eventJSONSchema)
	if err != nil {
		return
	}
	return statementList{
		{&s.insertEventJSONStmt, insertEventJSONSQL},
		{&s.bulkSelectEventJSONStmt, bulkSelectEventJSONSQL},
	}.prepare(db)
}

func (s *eventJSONStatements) insertEventJSON(
	ctx context.Context, eventNID types.EventNID, eventJSON []byte,
) error {
	_, err := s.insertEventJSONStmt.ExecContext(ctx, int64(eventNID), eventJSON)
	return err
}

type eventJSONPair struct {
	EventNID  types.EventNID
	EventJSON []byte
}

func (s *eventJSONStatements) bulkSelectEventJSON(
	ctx context.Context, eventNIDs []types.EventNID,
) ([]eventJSONPair, error) {
	rows, err := s.bulkSelectEventJSONStmt.QueryContext(ctx, eventNIDsAsArray(eventNIDs))
	if err != nil {
		fmt.Println("bulkSelectEventJSON s.bulkSelectEventJSONStmt.QueryContext:", err)
		return nil, err
	}
	defer rows.Close() // nolint: errcheck

	// We know that we will only get as many results as event NIDs
	// because of the unique constraint on event NIDs.
	// So we can allocate an array of the correct size now.
	// We might get fewer results than NIDs so we adjust the length of the slice before returning it.
	results := make([]eventJSONPair, len(eventNIDs))
	i := 0
	for ; rows.Next(); i++ {
		result := &results[i]
		var eventNID int64
		if err := rows.Scan(&eventNID, &result.EventJSON); err != nil {
			fmt.Println("bulkSelectEventJSON rows.Scan:", err)
			return nil, err
		}
		result.EventNID = types.EventNID(eventNID)
	}
	return results[:i], nil
}