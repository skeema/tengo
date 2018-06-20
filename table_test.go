package tengo

import (
	"fmt"
	"strings"
	"testing"
)

func TestTableGeneratedCreateStatement(t *testing.T) {
	for nextAutoInc := uint64(1); nextAutoInc < 3; nextAutoInc++ {
		table := aTable(nextAutoInc)
		if table.GeneratedCreateStatement() != table.CreateStatement {
			t.Errorf("Generated DDL does not match actual DDL\nExpected:\n%s\nFound:\n%s", table.CreateStatement, table.GeneratedCreateStatement())
		}
	}

	table := anotherTable()
	if table.GeneratedCreateStatement() != table.CreateStatement {
		t.Errorf("Generated DDL does not match actual DDL\nExpected:\n%s\nFound:\n%s", table.CreateStatement, table.GeneratedCreateStatement())
	}

	table = unsupportedTable()
	if table.GeneratedCreateStatement() == table.CreateStatement {
		t.Error("Expected unsupported table's generated DDL to differ from actual DDL, but they match")
	}
}

func TestClusteredIndexKey(t *testing.T) {
	table := aTable(1)
	if table.ClusteredIndexKey() == nil || table.ClusteredIndexKey() != table.PrimaryKey {
		t.Error("ClusteredIndexKey() did not return primary key when it was supposed to")
	}
	table.Engine = "MyISAM"
	if table.ClusteredIndexKey() != nil {
		t.Errorf("Expected ClusteredIndexKey() to return nil for non-InnoDB table, instead found %+v", table.ClusteredIndexKey())
	}
	table.Engine = "InnoDB"

	table.PrimaryKey = nil
	if table.ClusteredIndexKey() != table.SecondaryIndexes[0] {
		t.Errorf("Expected ClusteredIndexKey() to return %+v, instead found %+v", table.SecondaryIndexes[0], table.ClusteredIndexKey())
	}

	table.SecondaryIndexes[0], table.SecondaryIndexes[1] = table.SecondaryIndexes[1], table.SecondaryIndexes[0]
	if table.ClusteredIndexKey() != table.SecondaryIndexes[1] {
		t.Errorf("Expected ClusteredIndexKey() to return %+v, instead found %+v", table.SecondaryIndexes[1], table.ClusteredIndexKey())
	}

	table.Columns[4].Nullable = true
	if table.ClusteredIndexKey() != nil {
		t.Errorf("Expected ClusteredIndexKey() to return nil for table with unique-but-nullable index, instead found %+v", table.ClusteredIndexKey())
	}
	table.Columns[4].Nullable = false

	table.SecondaryIndexes[0].Unique = true
	if table.ClusteredIndexKey() != table.SecondaryIndexes[1] {
		t.Errorf("Expected ClusteredIndexKey() to return %+v, instead found %+v", table.SecondaryIndexes[1], table.ClusteredIndexKey())
	}

	table.Columns[2].Nullable = false
	if table.ClusteredIndexKey() != table.SecondaryIndexes[0] {
		t.Errorf("Expected ClusteredIndexKey() to return %+v, instead found %+v", table.SecondaryIndexes[0], table.ClusteredIndexKey())
	}
}

func TestTableAlterAddOrDropColumn(t *testing.T) {
	from := aTable(1)
	to := aTable(1)

	// Add a column to an arbitrary position
	newCol := &Column{
		Name:     "age",
		TypeInDB: "int unsigned",
		Nullable: true,
		Default:  ColumnDefaultNull,
	}
	to.Columns = append(to.Columns, newCol)
	colCount := len(to.Columns)
	to.Columns[colCount-2], to.Columns[colCount-1] = to.Columns[colCount-1], to.Columns[colCount-2]
	to.CreateStatement = to.GeneratedCreateStatement()
	tableAlters, supported := from.Diff(&to)
	if len(tableAlters) != 1 || !supported {
		t.Fatalf("Incorrect number of table alters: expected 1, found %d", len(tableAlters))
	}
	ta, ok := tableAlters[0].(AddColumn)
	if !ok {
		t.Fatalf("Incorrect type of table alter returned: expected %T, found %T", ta, tableAlters[0])
	}
	if ta.Table != &to || ta.Column != newCol {
		t.Error("Pointers in table alter do not point to expected values")
	}
	if ta.PositionFirst || ta.PositionAfter != to.Columns[colCount-3] {
		t.Errorf("Expected new column to be after `%s` / first=false, instead found after `%s` / first=%t", to.Columns[colCount-3].Name, ta.PositionAfter.Name, ta.PositionFirst)
	}

	// Reverse comparison should yield a drop-column
	tableAlters, supported = to.Diff(&from)
	if len(tableAlters) != 1 || !supported {
		t.Fatalf("Incorrect number of table alters: expected 1, found %d", len(tableAlters))
	}
	ta2, ok := tableAlters[0].(DropColumn)
	if !ok {
		t.Fatalf("Incorrect type of table alter returned: expected %T, found %T", ta2, tableAlters[0])
	}
	if ta2.Table != &to || ta2.Column != newCol {
		t.Error("Pointers in table alter do not point to expected values")
	}

	// Add an addition column to first position
	hadColumns := to.Columns
	anotherCol := &Column{
		Name:     "net_worth",
		TypeInDB: "decimal(9,2)",
		Nullable: true,
		Default:  ColumnDefaultNull,
	}
	to.Columns = []*Column{anotherCol}
	to.Columns = append(to.Columns, hadColumns...)
	to.CreateStatement = to.GeneratedCreateStatement()
	tableAlters, supported = from.Diff(&to)
	if len(tableAlters) != 2 || !supported {
		t.Fatalf("Incorrect number of table alters: expected 2, found %d", len(tableAlters))
	}
	ta, ok = tableAlters[0].(AddColumn)
	if !ok {
		t.Fatalf("Incorrect type of table alter[0] returned: expected %T, found %T", ta, tableAlters[0])
	}
	if ta.Table != &to || ta.Column != anotherCol {
		t.Error("Pointers in table alter[0] do not point to expected values")
	}
	if !ta.PositionFirst || ta.PositionAfter != nil {
		t.Errorf("Expected first new column to be after nil / first=true, instead found after %v / first=%t", ta.PositionAfter, ta.PositionFirst)
	}

	// Add an additional column to the last position
	anotherCol = &Column{
		Name:     "awards_won",
		TypeInDB: "int unsigned",
		Nullable: false,
		Default:  ColumnDefaultValue("0"),
	}
	to.Columns = append(to.Columns, anotherCol)
	to.CreateStatement = to.GeneratedCreateStatement()
	tableAlters, supported = from.Diff(&to)
	if len(tableAlters) != 3 || !supported {
		t.Fatalf("Incorrect number of table alters: expected 3, found %d", len(tableAlters))
	}
	ta, ok = tableAlters[2].(AddColumn)
	if !ok {
		t.Fatalf("Incorrect type of table alter[2] returned: expected %T, found %T", ta, tableAlters[2])
	}
	if ta.Table != &to || ta.Column != anotherCol {
		t.Error("Pointers in table alter[2] do not point to expected values")
	}
	if ta.PositionFirst || ta.PositionAfter != nil {
		t.Errorf("Expected new column to be after nil / first=false, instead found after %v / first=%t", ta.PositionAfter, ta.PositionFirst)
	}
}

func TestTableAlterAddOrDropIndex(t *testing.T) {
	from := aTable(1)
	to := aTable(1)

	// Add a secondary index
	newSecondary := &Index{
		Name:     "idx_alive_lastname",
		Columns:  []*Column{to.Columns[5], to.Columns[2]},
		SubParts: []uint16{0, 10},
	}
	to.SecondaryIndexes = append(to.SecondaryIndexes, newSecondary)
	to.CreateStatement = to.GeneratedCreateStatement()
	tableAlters, supported := from.Diff(&to)
	if len(tableAlters) != 1 || !supported {
		t.Fatalf("Incorrect number of table alters: expected 1, found %d", len(tableAlters))
	}
	ta, ok := tableAlters[0].(AddIndex)
	if !ok {
		t.Fatalf("Incorrect type of table alter returned: expected %T, found %T", ta, tableAlters[0])
	}
	if ta.Table != &to || ta.Index != newSecondary {
		t.Error("Pointers in table alter do not point to expected values")
	}

	// Reverse comparison should yield a drop index
	tableAlters, supported = to.Diff(&from)
	if len(tableAlters) != 1 || !supported {
		t.Fatalf("Incorrect number of table alters: expected 1, found %d", len(tableAlters))
	}
	ta2, ok := tableAlters[0].(DropIndex)
	if !ok {
		t.Fatalf("Incorrect type of table alter returned: expected %T, found %T", ta2, tableAlters[0])
	}
	if ta2.Table != &from || ta2.Index != newSecondary {
		t.Error("Pointers in table alter do not point to expected values")
	}

	// Start over; change the last existing secondary index
	to = aTable(1)
	to.SecondaryIndexes[1].Unique = true
	to.CreateStatement = to.GeneratedCreateStatement()
	tableAlters, supported = from.Diff(&to)
	if len(tableAlters) != 2 || !supported {
		t.Fatalf("Incorrect number of table alters: expected 2, found %d", len(tableAlters))
	}
	ta2, ok = tableAlters[0].(DropIndex)
	if !ok {
		t.Fatalf("Incorrect type of table alter[0] returned: expected %T, found %T", ta2, tableAlters[0])
	}
	if ta2.Table != &to || ta2.Index != from.SecondaryIndexes[1] {
		t.Error("Pointers in table alter[0] do not point to expected values")
	}
	ta, ok = tableAlters[1].(AddIndex)
	if !ok {
		t.Fatalf("Incorrect type of table alter[1] returned: expected %T, found %T", ta, tableAlters[1])
	}
	if ta.Table != &to || ta.Index != to.SecondaryIndexes[1] {
		t.Error("Pointers in table alter[1] do not point to expected values")
	}

	// Start over; change the primary key
	to = aTable(1)
	to.PrimaryKey.Columns = append(to.PrimaryKey.Columns, to.Columns[4])
	to.PrimaryKey.SubParts = append(to.PrimaryKey.SubParts, 0)
	to.CreateStatement = to.GeneratedCreateStatement()
	tableAlters, supported = from.Diff(&to)
	if len(tableAlters) != 2 || !supported {
		t.Fatalf("Incorrect number of table alters: expected 2, found %d", len(tableAlters))
	}
	ta2, ok = tableAlters[0].(DropIndex)
	if !ok {
		t.Fatalf("Incorrect type of table alter[0] returned: expected %T, found %T", ta2, tableAlters[0])
	}
	if ta2.Table != &to || ta2.Index != from.PrimaryKey {
		t.Error("Pointers in table alter[0] do not point to expected values")
	}
	ta, ok = tableAlters[1].(AddIndex)
	if !ok {
		t.Fatalf("Incorrect type of table alter[1] returned: expected %T, found %T", ta, tableAlters[1])
	}
	if ta.Table != &to || ta.Index != to.PrimaryKey {
		t.Error("Pointers in table alter[1] do not point to expected values")
	}

	// Remove the primary key
	to.PrimaryKey = nil
	to.CreateStatement = to.GeneratedCreateStatement()
	tableAlters, supported = from.Diff(&to)
	if len(tableAlters) != 1 || !supported {
		t.Fatalf("Incorrect number of table alters: expected 1, found %d", len(tableAlters))
	}
	ta2, ok = tableAlters[0].(DropIndex)
	if !ok {
		t.Fatalf("Incorrect type of table alter returned: expected %T, found %T", ta2, tableAlters[0])
	}
	if ta2.Table != &to || ta2.Index != from.PrimaryKey {
		t.Error("Pointers in table alter do not point to expected values")
	}

	// Reverse comparison should yield an add PK
	tableAlters, supported = to.Diff(&from)
	if len(tableAlters) != 1 || !supported {
		t.Fatalf("Incorrect number of table alters: expected 1, found %d", len(tableAlters))
	}
	ta, ok = tableAlters[0].(AddIndex)
	if !ok {
		t.Fatalf("Incorrect type of table alter returned: expected %T, found %T", ta, tableAlters[0])
	}
	if ta.Table != &from || ta.Index != from.PrimaryKey {
		t.Error("Pointers in table alter do not point to expected values")
	}
}

func TestTableAlterAddIndexOrder(t *testing.T) {
	from := aTable(1)
	to := aTable(1)

	// Add 10 secondary indexes, and ensure their order is preserved
	for n := 0; n < 10; n++ {
		to.SecondaryIndexes = append(to.SecondaryIndexes, &Index{
			Name:     fmt.Sprintf("newidx_%d", n),
			Columns:  []*Column{to.Columns[0]},
			SubParts: []uint16{0},
		})
	}
	to.CreateStatement = to.GeneratedCreateStatement()
	tableAlters, supported := from.Diff(&to)
	if len(tableAlters) != 10 || !supported {
		t.Fatalf("Incorrect number of table alters: expected 10, found %d, supported=%t", len(tableAlters), supported)
	}
	for n := 0; n < 10; n++ {
		ta, ok := tableAlters[n].(AddIndex)
		if !ok {
			t.Fatalf("Incorrect type of table alter returned: expected %T, found %T", ta, tableAlters[0])
		}
		expectName := fmt.Sprintf("newidx_%d", n)
		if ta.Index.Name != expectName {
			t.Errorf("Incorrect index order: expected alters[%d] to be index name %s, instead found %s", n, expectName, ta.Index.Name)
		}
	}

	// Also modify an existing index, and ensure its corresponding drop + re-add
	// comes before the other 10 adds
	to.SecondaryIndexes[1].SubParts[1] = 6
	tableAlters, supported = from.Diff(&to)
	if len(tableAlters) != 12 || !supported {
		t.Fatalf("Incorrect number of table alters: expected 12, found %d, supported=%t", len(tableAlters), supported)
	}
	if ta, ok := tableAlters[0].(DropIndex); !ok {
		t.Errorf("Expected alters[0] to be a DropIndex, instead found %T", ta)
	}
	if ta, ok := tableAlters[1].(AddIndex); !ok {
		t.Errorf("Expected alters[1] to be an AddIndex, instead found %T", ta)
	} else if ta.Index.Name != to.SecondaryIndexes[1].Name {
		t.Errorf("Expected alters[1] to be on index %s, instead found %s", to.SecondaryIndexes[1].Name, ta.Index.Name)
	}
}

func TestTableAlterIndexReorder(t *testing.T) {
	// Table with three secondary indexes:
	// [0] is UNIQUE KEY `idx_ssn` (`ssn`)
	// [1] is KEY `idx_actor_name` (`last_name`(10),`first_name`(1))
	// [2] is KEY `idx_alive_lastname` (`alive`, `last_name`(10))
	getTable := func() Table {
		table := aTable(1)
		table.SecondaryIndexes = append(table.SecondaryIndexes, &Index{
			Name:     "idx_alive_lastname",
			Columns:  []*Column{table.Columns[5], table.Columns[2]},
			SubParts: []uint16{0, 10},
		})
		table.CreateStatement = table.GeneratedCreateStatement()
		return table
	}

	assertClauses := func(from, to *Table, strict bool, format string, a ...interface{}) {
		t.Helper()
		td := NewAlterTable(from, to)
		var clauses string
		if td != nil {
			var err error
			clauses, err = td.Clauses(StatementModifiers{
				StrictIndexOrder: strict,
			})
			if err != nil {
				t.Fatalf("Unexpected error result from Clauses(): %s", err)
			}
		}
		expected := fmt.Sprintf(format, a...)
		if clauses != expected {
			t.Errorf("Unexpected result from Clauses()\nExpected:\n  %s\nFound:\n  %s", expected, clauses)
		}
	}

	from, to := getTable(), getTable()
	orig := from.SecondaryIndexes

	// Reorder to's last couple indexes ([1] and [2]). Resulting diff should
	// drop [1] and re-add [1], but should manifest as a no-op statement unless
	// mods.StrictIndexOrder enabled.
	to.SecondaryIndexes[1], to.SecondaryIndexes[2] = to.SecondaryIndexes[2], to.SecondaryIndexes[1]
	to.CreateStatement = to.GeneratedCreateStatement()
	tableAlters, _ := from.Diff(&to)
	if len(tableAlters) != 2 {
		t.Errorf("Expected 2 clauses, instead found %d", len(tableAlters))
	} else {
		if drop, ok := tableAlters[0].(DropIndex); !ok {
			t.Errorf("Expected tableAlters[0] to be %T, instead found %T", drop, tableAlters[0])
		} else if drop.Index.Name != orig[1].Name {
			t.Errorf("Expected tableAlters[0] to drop %s, instead dropped %s", orig[1].Name, drop.Index.Name)
		}
		if add, ok := tableAlters[1].(AddIndex); !ok {
			t.Errorf("Expected tableAlters[1] to be %T, instead found %T", add, tableAlters[1])
		} else if add.Index.Name != orig[1].Name {
			t.Errorf("Expected tableAlters[1] to add %s, instead added %s", orig[1].Name, add.Index.Name)
		}
		assertClauses(&from, &to, false, "")
		assertClauses(&from, &to, true, "DROP INDEX `%s`, ADD %s", orig[1].Name, orig[1].Definition())
	}

	// Clustered index key changes: same effect as mods.StrictIndexOrder
	to.PrimaryKey = nil
	to.CreateStatement = to.GeneratedCreateStatement()
	assertClauses(&from, &to, false, "DROP PRIMARY KEY, DROP INDEX `%s`, ADD %s", orig[1].Name, orig[1].Definition())
	assertClauses(&from, &to, true, "DROP PRIMARY KEY, DROP INDEX `%s`, ADD %s", orig[1].Name, orig[1].Definition())

	// Restore to previous state, and then modify [1]. Resulting diff should drop
	// [1] and [2], then re-add the modified [1], and then re-add the unmodified
	// [2]. Corresponding statement should only refer to [1] unless
	// mods.StrictIndexOrder used.
	to = getTable()
	to.SecondaryIndexes[1].SubParts[1] = 8
	to.CreateStatement = to.GeneratedCreateStatement()
	tableAlters, _ = from.Diff(&to)
	if len(tableAlters) != 4 {
		t.Errorf("Expected 4 clauses, instead found %d", len(tableAlters))
	} else {
		drop1, ok1 := tableAlters[0].(DropIndex)
		drop2, ok2 := tableAlters[1].(DropIndex)
		add3, ok3 := tableAlters[2].(AddIndex)
		add4, ok4 := tableAlters[3].(AddIndex)
		if !ok1 || !ok2 || !ok3 || !ok4 {
			t.Errorf("One or more type mismatches; ok: %t %t %t %t", ok1, ok2, ok3, ok4)
		} else {
			if drop1.Index.Name == drop2.Index.Name {
				t.Errorf("Both drops refer to same index %s", drop1.Index.Name)
			}
			if add3.Index.Name != orig[1].Name || add3.Index.SubParts[1] != 8 {
				t.Errorf("tableAlters[2] does not match expectations; found %+v", add3.Index)
			}
			if !add4.Index.Equals(orig[2]) {
				t.Errorf("tableAlters[3] does not match expectations; found %+v", add4.Index)
			}
		}
		assertClauses(&from, &to, false, "DROP INDEX `%s`, ADD %s", orig[1].Name, to.SecondaryIndexes[1].Definition())
		assertClauses(&from, &to, true, "DROP INDEX `%s`, DROP INDEX `%s`, ADD %s, ADD %s", orig[1].Name, orig[2].Name, to.SecondaryIndexes[1].Definition(), orig[2].Definition())
	}

	// Adding a new index before [1] should also result in dropping the old [1]
	// and [2], and then re-adding them back in that order. But statement should
	// only refer to adding the new index unless mods.StrictIndexOrder used.
	to = getTable()
	newIdx := &Index{
		Name:     "idx_firstname",
		Columns:  []*Column{to.Columns[1]},
		SubParts: []uint16{0},
	}
	to.SecondaryIndexes = []*Index{to.SecondaryIndexes[0], newIdx, to.SecondaryIndexes[1], to.SecondaryIndexes[2]}
	to.CreateStatement = to.GeneratedCreateStatement()
	tableAlters, _ = from.Diff(&to)
	if len(tableAlters) != 5 {
		t.Errorf("Expected 5 clauses, instead found %d", len(tableAlters))
	} else {
		assertClauses(&from, &to, false, "ADD %s", newIdx.Definition())
		assertClauses(&from, &to, true, "DROP INDEX `%s`, DROP INDEX `%s`, ADD %s, ADD %s, ADD %s", orig[1].Name, orig[2].Name, newIdx.Definition(), orig[1].Definition(), orig[2].Definition())
	}

	// The opposite operation -- dropping the new index that we put before [1] --
	// should just result in a drop, no need to reorder anything
	tableAlters, _ = to.Diff(&from)
	if len(tableAlters) != 1 {
		t.Errorf("Expected 1 clause, instead found %d", len(tableAlters))
	} else {
		if drop, ok := tableAlters[0].(DropIndex); !ok {
			t.Errorf("Expected tableAlters[0] to be %T, instead found %T", drop, tableAlters[0])
		} else if drop.Index.Name != newIdx.Name {
			t.Errorf("Expected tableAlters[0] to drop %s, instead dropped %s", newIdx.Name, drop.Index.Name)
		}
		assertClauses(&to, &from, false, "DROP INDEX `%s`", newIdx.Name)
		assertClauses(&to, &from, true, "DROP INDEX `%s`", newIdx.Name)
	}
}

func TestTableAlterModifyColumn(t *testing.T) {
	from := aTable(1)
	to := aTable(1)

	// Reposition a col to first position
	movedColPos := 3
	movedCol := to.Columns[movedColPos]
	to.Columns = append(to.Columns[:movedColPos], to.Columns[movedColPos+1:]...)
	to.Columns = append([]*Column{movedCol}, to.Columns...)
	to.CreateStatement = to.GeneratedCreateStatement()
	tableAlters, supported := from.Diff(&to)
	if len(tableAlters) != 1 || !supported {
		t.Fatalf("Incorrect number of table alters: expected 1, found %d", len(tableAlters))
	}
	ta, ok := tableAlters[0].(ModifyColumn)
	if !ok {
		t.Fatalf("Incorrect type of table alter returned: expected %T, found %T", ta, tableAlters[0])
	}
	if ta.Table != &to || ta.OldColumn != from.Columns[movedColPos] || ta.NewColumn != movedCol {
		t.Error("Pointers in table alter do not point to expected values")
	}
	if !ta.PositionFirst || ta.PositionAfter != nil {
		t.Errorf("Expected modified column to be after nil / first=true, instead found after %v / first=%t", ta.PositionAfter, ta.PositionFirst)
	}

	// Reposition same col to last position
	to = aTable(1)
	movedCol = to.Columns[movedColPos]
	shouldBeAfter := to.Columns[len(to.Columns)-1]
	to.Columns = append(to.Columns[:movedColPos], to.Columns[movedColPos+1:]...)
	to.Columns = append(to.Columns, movedCol)
	to.CreateStatement = to.GeneratedCreateStatement()
	tableAlters, supported = from.Diff(&to)
	if len(tableAlters) != 1 || !supported {
		t.Fatalf("Incorrect number of table alters: expected 1, found %d", len(tableAlters))
	}
	ta, ok = tableAlters[0].(ModifyColumn)
	if !ok {
		t.Fatalf("Incorrect type of table alter returned: expected %T, found %T", ta, tableAlters[0])
	}
	if ta.PositionFirst || ta.PositionAfter != shouldBeAfter {
		t.Errorf("Expected modified column to be after %s / first=false, instead found after %v / first=%t", shouldBeAfter.Name, ta.PositionAfter, ta.PositionFirst)
	}
	if !ta.NewColumn.Equals(ta.OldColumn) {
		t.Errorf("Column definition unexpectedly changed: was %s, now %s", ta.OldColumn.Definition(nil), ta.NewColumn.Definition(nil))
	}

	// Repos to last position AND change column definition
	movedCol.Nullable = !movedCol.Nullable
	to.CreateStatement = to.GeneratedCreateStatement()
	tableAlters, supported = from.Diff(&to)
	if len(tableAlters) != 1 || !supported {
		t.Fatalf("Incorrect number of table alters: expected 1, found %d", len(tableAlters))
	}
	ta, ok = tableAlters[0].(ModifyColumn)
	if !ok {
		t.Fatalf("Incorrect type of table alter returned: expected %T, found %T", ta, tableAlters[0])
	}
	if ta.PositionFirst || ta.PositionAfter != shouldBeAfter {
		t.Errorf("Expected modified column to be after %s / first=false, instead found after %v / first=%t", shouldBeAfter.Name, ta.PositionAfter, ta.PositionFirst)
	}
	if ta.NewColumn.Equals(ta.OldColumn) {
		t.Errorf("Column definition unexpectedly NOT changed: still %s", ta.NewColumn.Definition(nil))
	}

	// Start over; delete a col, move last col to its former position, and add a new col after that
	// FROM: actor_id, first_name, last_name, last_updated, ssn, alive, alive_bit
	// TO:   actor_id, first_name, last_name, alive, alive_bit, age, ssn
	// current move algo treats this as a move of ssn to be after alive, rather than alive to be after last_name
	to = aTable(1)
	newCol := &Column{
		Name:     "age",
		TypeInDB: "int unsigned",
		Nullable: true,
		Default:  ColumnDefaultNull,
	}
	to.Columns = append(to.Columns[0:3], to.Columns[5], to.Columns[6], newCol, to.Columns[4])
	to.CreateStatement = to.GeneratedCreateStatement()
	tableAlters, supported = from.Diff(&to)
	if len(tableAlters) != 3 || !supported {
		t.Fatalf("Incorrect number of table alters: expected 3, found %d", len(tableAlters))
	}
	// The alters should always be in this order: drops, modifications, adds
	if drop, ok := tableAlters[0].(DropColumn); ok {
		if drop.Table != &from || drop.Column != from.Columns[3] {
			t.Error("Pointers in table alter[0] do not point to expected values")
		}
	} else {
		t.Errorf("Incorrect type of table alter[0] returned: expected %T, found %T", drop, tableAlters[0])
	}
	if modify, ok := tableAlters[1].(ModifyColumn); ok {
		if modify.NewColumn.Name != "ssn" {
			t.Error("Pointers in table alter[1] do not point to expected values")
		}
		if modify.PositionFirst || modify.PositionAfter.Name != "alive_bit" {
			t.Errorf("Expected moved column to be after alive_bit / first=false, instead found after %v / first=%t", modify.PositionAfter, modify.PositionFirst)
		}
	} else {
		t.Errorf("Incorrect type of table alter[1] returned: expected %T, found %T", modify, tableAlters[1])
	}
	if add, ok := tableAlters[2].(AddColumn); ok {
		if add.PositionFirst || add.PositionAfter.Name != "alive_bit" {
			t.Errorf("Expected new column to be after alive_bit / first=false, instead found after %v / first=%t", add.PositionAfter, add.PositionFirst)
		}
	} else {
		t.Errorf("Incorrect type of table alter[2] returned: expected %T, found %T", add, tableAlters[2])
	}

	// Start over; just change a column definition without moving anything
	to = aTable(1)
	to.Columns[4].TypeInDB = "varchar(10)"
	to.CreateStatement = to.GeneratedCreateStatement()
	tableAlters, supported = from.Diff(&to)
	if len(tableAlters) != 1 || !supported {
		t.Fatalf("Incorrect number of table alters: expected 1, found %d", len(tableAlters))
	}
	ta, ok = tableAlters[0].(ModifyColumn)
	if !ok {
		t.Fatalf("Incorrect type of table alter returned: expected %T, found %T", ta, tableAlters[0])
	}
	if ta.PositionFirst || ta.PositionAfter != nil {
		t.Errorf("Expected modified column to not be moved, instead found after %v / first=%t", ta.PositionAfter, ta.PositionFirst)
	}
	if ta.NewColumn.Equals(from.Columns[4]) {
		t.Errorf("Column definition unexpectedly NOT changed: still %s", ta.NewColumn.Definition(nil))
	}

	// TODO: once the column-move algorithm is optimal, add a test that confirms
}

func TestTableAlterChangeStorageEngine(t *testing.T) {
	getTableWithEngine := func(engine string) Table {
		t := aTable(1)
		t.Engine = engine
		t.CreateStatement = t.GeneratedCreateStatement()
		return t
	}
	assertChangeEngine := func(a, b *Table, expected string) {
		tableAlters, supported := a.Diff(b)
		if expected == "" {
			if len(tableAlters) != 0 || !supported {
				t.Fatalf("Incorrect result from Table.Diff(): expected len=0, true; found len=%d, %t", len(tableAlters), supported)
			}
			return
		}
		if len(tableAlters) != 1 || !supported {
			t.Fatalf("Incorrect result from Table.Diff(): expected len=1, supported=true; found len=%d, supported=%t", len(tableAlters), supported)
		}
		ta, ok := tableAlters[0].(ChangeStorageEngine)
		if !ok {
			t.Fatalf("Incorrect type of table alter returned: expected %T, found %T", ta, tableAlters[0])
		}
		if ta.Clause() != expected {
			t.Errorf("Incorrect ALTER TABLE clause returned; expected: %s; found: %s", expected, ta.Clause())
		}
	}

	from := getTableWithEngine("InnoDB")
	to := getTableWithEngine("InnoDB")
	assertChangeEngine(&from, &to, "")
	to = getTableWithEngine("MyISAM")
	assertChangeEngine(&from, &to, "ENGINE=MyISAM")
	assertChangeEngine(&to, &from, "ENGINE=InnoDB")
}

func TestTableAlterChangeAutoIncrement(t *testing.T) {
	// Initial test: change next auto inc from 1 to 2
	from := aTable(1)
	to := aTable(2)
	tableAlters, supported := from.Diff(&to)
	if len(tableAlters) != 1 || !supported {
		t.Fatalf("Incorrect number of table alters: expected 1, found %d", len(tableAlters))
	}
	ta, ok := tableAlters[0].(ChangeAutoIncrement)
	if !ok {
		t.Fatalf("Incorrect type of table alter returned: expected %T, found %T", ta, tableAlters[0])
	}
	if ta.OldNextAutoIncrement != from.NextAutoIncrement || ta.NewNextAutoIncrement != to.NextAutoIncrement {
		t.Error("Incorrect next-auto-increment values in alter clause")
	}

	// Reverse test: should emit an alter clause, even though higher-level caller
	// may decide to ignore it
	tableAlters, supported = to.Diff(&from)
	if len(tableAlters) != 1 || !supported {
		t.Fatalf("Incorrect number of table alters: expected 1, found %d", len(tableAlters))
	}
	ta, ok = tableAlters[0].(ChangeAutoIncrement)
	if !ok {
		t.Fatalf("Incorrect type of table alter returned: expected %T, found %T", ta, tableAlters[0])
	}
	if ta.OldNextAutoIncrement != to.NextAutoIncrement || ta.NewNextAutoIncrement != from.NextAutoIncrement {
		t.Error("Incorrect next-auto-increment values in alter clause")
	}

	// Removing an auto-inc col and changing auto inc next value: should NOT emit
	// an auto-inc change since "to" table no longer has one
	to.Columns = to.Columns[1:]
	to.CreateStatement = to.GeneratedCreateStatement()
	tableAlters, supported = from.Diff(&to)
	if len(tableAlters) != 1 || !supported {
		t.Errorf("Incorrect number of table alters: expected 1, found %d", len(tableAlters))
	} else {
		ta, ok := tableAlters[0].(DropColumn)
		if !ok {
			t.Fatalf("Incorrect type of table alter returned: expected %T, found %T", ta, tableAlters[0])
		}
	}

	// Reverse of above comparison, adding an auto-inc col with a non-default
	// starting value: one clause for new col, another for auto-inc val (in that order!)
	to.NextAutoIncrement = 0
	from.NextAutoIncrement = 3
	to.CreateStatement = to.GeneratedCreateStatement()
	from.CreateStatement = from.GeneratedCreateStatement()
	tableAlters, supported = to.Diff(&from)
	if len(tableAlters) != 2 || !supported {
		t.Errorf("Incorrect number of table alters: expected 2, found %d", len(tableAlters))
	} else {
		if ta, ok := tableAlters[0].(AddColumn); !ok {
			t.Fatalf("Incorrect type of table alter[0] returned: expected %T, found %T", ta, tableAlters[0])
		}
		if ta, ok := tableAlters[1].(ChangeAutoIncrement); !ok {
			t.Fatalf("Incorrect type of table alter[1] returned: expected %T, found %T", ta, tableAlters[1])
		}
	}
}

func TestTableAlterChangeCharSet(t *testing.T) {
	getTableWithCharSet := func(charSet, collation string) Table {
		t := aTable(1)
		t.CharSet = charSet
		t.Collation = collation
		t.CreateStatement = t.GeneratedCreateStatement()
		return t
	}
	assertChangeCharSet := func(a, b *Table, expected string) {
		tableAlters, supported := a.Diff(b)
		if expected == "" {
			if len(tableAlters) != 0 || !supported {
				t.Fatalf("Incorrect result from Table.Diff(): expected len=0, true; found len=%d, %t", len(tableAlters), supported)
			}
			return
		}
		if len(tableAlters) != 1 || !supported {
			t.Fatalf("Incorrect result from Table.Diff(): expected len=1, supported=true; found len=%d, supported=%t", len(tableAlters), supported)
		}
		ta, ok := tableAlters[0].(ChangeCharSet)
		if !ok {
			t.Fatalf("Incorrect type of table alter returned: expected %T, found %T", ta, tableAlters[0])
		}
		if ta.Clause() != expected {
			t.Errorf("Incorrect ALTER TABLE clause returned; expected: %s; found: %s", expected, ta.Clause())
		}
	}

	from := getTableWithCharSet("utf8mb4", "")
	to := getTableWithCharSet("utf8mb4", "")
	assertChangeCharSet(&from, &to, "")

	to = getTableWithCharSet("utf8mb4", "utf8mb4_swedish_ci")
	assertChangeCharSet(&from, &to, "DEFAULT CHARACTER SET = utf8mb4 COLLATE = utf8mb4_swedish_ci")
	assertChangeCharSet(&to, &from, "DEFAULT CHARACTER SET = utf8mb4")

	to = getTableWithCharSet("latin1", "")
	assertChangeCharSet(&from, &to, "DEFAULT CHARACTER SET = latin1")
	assertChangeCharSet(&to, &from, "DEFAULT CHARACTER SET = utf8mb4")

	to = getTableWithCharSet("latin1", "latin1_general_ci")
	assertChangeCharSet(&from, &to, "DEFAULT CHARACTER SET = latin1 COLLATE = latin1_general_ci")
	assertChangeCharSet(&to, &from, "DEFAULT CHARACTER SET = utf8mb4")
}

func TestTableAlterChangeCreateOptions(t *testing.T) {
	getTableWithCreateOptions := func(createOptions string) Table {
		t := aTable(1)
		t.CreateOptions = createOptions
		t.CreateStatement = t.GeneratedCreateStatement()
		return t
	}
	assertChangeCreateOptions := func(a, b *Table, expected string) {
		tableAlters, supported := a.Diff(b)
		if expected == "" {
			if len(tableAlters) != 0 || !supported {
				t.Fatalf("Incorrect result from Table.Diff(): expected len=0, true; found len=%d, %t", len(tableAlters), supported)
			}
			return
		}
		if len(tableAlters) != 1 || !supported {
			t.Fatalf("Incorrect result from Table.Diff(): expected len=1, supported=true; found len=%d, supported=%t", len(tableAlters), supported)
		}
		ta, ok := tableAlters[0].(ChangeCreateOptions)
		if !ok {
			t.Fatalf("Incorrect type of table alter returned: expected %T, found %T", ta, tableAlters[0])
		}

		// Order of result isn't predictable, so convert to maps and compare
		indexedClause := make(map[string]bool)
		indexedExpected := make(map[string]bool)
		for _, token := range strings.Split(ta.Clause(), " ") {
			indexedClause[token] = true
		}
		for _, token := range strings.Split(expected, " ") {
			indexedExpected[token] = true
		}

		if len(indexedClause) != len(indexedExpected) {
			t.Errorf("Incorrect ALTER TABLE clause returned; expected: %s; found: %s", expected, ta.Clause())
			return
		}
		for k, v := range indexedExpected {
			if foundv, ok := indexedClause[k]; v != foundv || !ok {
				t.Errorf("Incorrect ALTER TABLE clause returned; expected: %s; found: %s", expected, ta.Clause())
				return
			}
		}
	}

	from := getTableWithCreateOptions("")
	to := getTableWithCreateOptions("")
	assertChangeCreateOptions(&from, &to, "")

	to = getTableWithCreateOptions("ROW_FORMAT=DYNAMIC")
	assertChangeCreateOptions(&from, &to, "ROW_FORMAT=DYNAMIC")
	assertChangeCreateOptions(&to, &from, "ROW_FORMAT=DEFAULT")

	to = getTableWithCreateOptions("STATS_PERSISTENT=1 ROW_FORMAT=DYNAMIC")
	assertChangeCreateOptions(&from, &to, "STATS_PERSISTENT=1 ROW_FORMAT=DYNAMIC")
	assertChangeCreateOptions(&to, &from, "STATS_PERSISTENT=DEFAULT ROW_FORMAT=DEFAULT")

	from = getTableWithCreateOptions("ROW_FORMAT=REDUNDANT AVG_ROW_LENGTH=200 STATS_PERSISTENT=1 MAX_ROWS=1000")
	to = getTableWithCreateOptions("STATS_AUTO_RECALC=1 ROW_FORMAT=DYNAMIC AVG_ROW_LENGTH=200")
	assertChangeCreateOptions(&from, &to, "STATS_AUTO_RECALC=1 ROW_FORMAT=DYNAMIC STATS_PERSISTENT=DEFAULT MAX_ROWS=0")
	assertChangeCreateOptions(&to, &from, "STATS_AUTO_RECALC=DEFAULT ROW_FORMAT=REDUNDANT STATS_PERSISTENT=1 MAX_ROWS=1000")
}

func TestTableAlterChangeComment(t *testing.T) {
	getTableWithComment := func(comment string) Table {
		t := aTable(1)
		t.Comment = comment
		t.CreateStatement = t.GeneratedCreateStatement()
		return t
	}
	assertChangeComment := func(a, b *Table, expected string) {
		tableAlters, supported := a.Diff(b)
		if expected == "" {
			if len(tableAlters) != 0 || !supported {
				t.Fatalf("Incorrect result from Table.Diff(): expected len=0, true; found len=%d, %t", len(tableAlters), supported)
			}
			return
		}
		if len(tableAlters) != 1 || !supported {
			t.Fatalf("Incorrect result from Table.Diff(): expected len=1, supported=true; found len=%d, supported=%t", len(tableAlters), supported)
		}
		ta, ok := tableAlters[0].(ChangeComment)
		if !ok {
			t.Fatalf("Incorrect type of table alter returned: expected %T, found %T", ta, tableAlters[0])
		}
		if ta.Clause() != expected {
			t.Errorf("Incorrect ALTER TABLE clause returned; expected: %s; found: %s", expected, ta.Clause())
		}
	}

	from := getTableWithComment("")
	to := getTableWithComment("")
	assertChangeComment(&from, &to, "")
	to = getTableWithComment("I'm a table-level comment!")
	assertChangeComment(&from, &to, "COMMENT 'I''m a table-level comment!'")
	assertChangeComment(&to, &from, "COMMENT ''")
}

func TestTableAlterUnsupportedTable(t *testing.T) {
	from, to := unsupportedTable(), unsupportedTable()
	newCol := &Column{
		Name:     "age",
		TypeInDB: "int unsigned",
		Nullable: true,
		Default:  ColumnDefaultNull,
	}
	to.Columns = append(to.Columns, newCol)
	to.CreateStatement = to.GeneratedCreateStatement()
	if tableAlters, supported := from.Diff(&to); len(tableAlters) != 0 || supported {
		t.Fatalf("Expected diff of unsupported tables to yield no alters; instead found %d", len(tableAlters))
	}

	// Confirm same behavior even if only one side is marked as unsupported
	from, to = anotherTable(), unsupportedTable()
	from.Name = to.Name
	if tableAlters, supported := from.Diff(&to); len(tableAlters) != 0 || supported {
		t.Fatalf("Expected diff of unsupported tables to yield no alters; instead found %d", len(tableAlters))
	}
	from, to = to, from
	if tableAlters, supported := from.Diff(&to); len(tableAlters) != 0 || supported {
		t.Fatalf("Expected diff of unsupported tables to yield no alters; instead found %d", len(tableAlters))
	}
}
