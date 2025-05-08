package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
	_ "modernc.org/sqlite"
)

type Column struct {
	Name string
	Type string // VARCHAR or DATE
}

func main() {
	// CLI flags for batch mode
	batch := flag.Bool("batch-merge", false, "Run batch mail merge")
	template := flag.String("template", "", "Template file")
	dbFile := flag.String("db", "", "SQLite database")
	output := flag.String("output", "", "Output folder")
	flag.Parse()

	if *batch {
		if *template == "" || *dbFile == "" || *output == "" {
			log.Fatal("Missing required flags: --template, --db, --output")
		}
		db, err := sql.Open("sqlite", *dbFile)
		if err != nil {
			log.Fatal("Failed to open database:", err)
		}
		defer db.Close()
		mailMerge(db, *template, *output)
		return
	}

	// Initialize SQLite database for GUI
	db, err := sql.Open("sqlite", "goatpad.db")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	gtk.Init(nil)

	// Load Glade file
	builder, err := gtk.BuilderNewFromFile("goatpad.glade")
	if err != nil {
		log.Fatal("Failed to load Glade file:", err)
	}

	// Get main window
	win, err := builder.GetObject("main_window")
	if err != nil {
		log.Fatal("Failed to get window:", err)
	}
	window := win.(*gtk.Window)

	// Get text view and buffer
	tvObj, err := builder.GetObject("text_view")
	if err != nil {
		log.Fatal("Failed to get text view:", err)
	}
	textView := tvObj.(*gtk.TextView)
	buffer, err := textView.GetBuffer()
	if err != nil {
		log.Fatal("Failed to get buffer:", err)
	}

	// Create text tags
	tagTable, _ := buffer.GetTagTable()
	boldTag, _ := gtk.TextTagNew("bold")
	boldTag.SetProperty("weight", pango.WEIGHT_BOLD)
	tagTable.Add(boldTag)
	italicTag, _ := gtk.TextTagNew("italic")
	italicTag.SetProperty("style", pango.STYLE_ITALIC)
	tagTable.Add(italicTag)
	sizeTag, _ := gtk.TextTagNew("size")
	sizeTag.SetProperty("size-points", 12.0)
	tagTable.Add(sizeTag)
	leftTag, _ := gtk.TextTagNew("left")
	leftTag.SetProperty("justification", gtk.JUSTIFY_LEFT)
	tagTable.Add(leftTag)
	centerTag, _ := gtk.TextTagNew("center")
	centerTag.SetProperty("justification", gtk.JUSTIFY_CENTER)
	tagTable.Add(centerTag)

	// Bold button
	boldBtnObj, _ := builder.GetObject("bold_button")
	boldBtn := boldBtnObj.(*gtk.ToolButton)
	boldBtn.Connect("clicked", func() {
		start, end, ok := buffer.GetSelectionBounds()
		if ok {
			buffer.ApplyTag(boldTag, start, end)
		}
	})

	// Italic button
	italicBtnObj, _ := builder.GetObject("italic_button")
	italicBtn := italicBtnObj.(*gtk.ToolButton)
	italicBtn.Connect("clicked", func() {
		start, end, ok := buffer.GetSelectionBounds()
		if ok {
			buffer.ApplyTag(italicTag, start, end)
		}
	})

	// Font size combo
	fontSizeObj, _ := builder.GetObject("font_size_combo")
	fontSizeCombo := fontSizeObj.(*gtk.ComboBoxText)
	fontSizeCombo.Connect("changed", func() {
		size, _ := strconv.ParseFloat(fontSizeCombo.GetActiveText(), 64)
		sizeTag.SetProperty("size-points", size)
		start, end, ok := buffer.GetSelectionBounds()
		if ok {
			buffer.ApplyTag(sizeTag, start, end)
		}
	})

	// Align left button
	alignLeftBtnObj, _ := builder.GetObject("align_left_button")
	alignLeftBtn := alignLeftBtnObj.(*gtk.ToolButton)
	alignLeftBtn.Connect("clicked", func() {
		start, end, ok := buffer.GetSelectionBounds()
		if ok {
			buffer.ApplyTag(leftTag, start, end)
		}
	})

	// Align center button
	alignCenterBtnObj, _ := builder.GetObject("align_center_button")
	alignCenterBtn := alignCenterBtnObj.(*gtk.ToolButton)
	alignCenterBtn.Connect("clicked", func() {
		start, end, ok := buffer.GetSelectionBounds()
		if ok {
			buffer.ApplyTag(centerTag, start, end)
		}
	})

	// Save button (async)
	saveBtnObj, _ := builder.GetObject("save_button")
	saveBtn := saveBtnObj.(*gtk.ToolButton)
	saveBtn.Connect("clicked", func() {
		dialog, err := gtk.FileChooserDialogNewWith2Buttons(
			"Save File", window, gtk.FILE_CHOOSER_ACTION_SAVE,
			"Cancel", gtk.RESPONSE_CANCEL,
			"Save", gtk.RESPONSE_ACCEPT,
		)
		if err != nil {
			log.Fatal("Unable to create file chooser dialog:", err)
		}
		filter, _ := gtk.FileFilterNew()
		filter.AddPattern("*.txt")
		filter.AddPattern("*.rtf")
		dialog.AddFilter(filter)
		if dialog.Run() == gtk.RESPONSE_ACCEPT {
			filename := dialog.GetFilename()
			start, end := buffer.GetBounds()
			text, _ := buffer.GetText(start, end, false)
			go func() {
				if strings.HasSuffix(filename, ".rtf") {
					rtf := "{\\rtf1\\ansi\\deff0\\fonttbl{\\f0 Arial;}\\fs24 " + text + "}"
					os.WriteFile(filename, []byte(rtf), 0644)
				} else {
					file, err := os.Create(filename)
					if err != nil {
						log.Println("Save error:", err)
						return
					}
					defer file.Close()
					_, err = io.WriteString(file, text)
					if err != nil {
						log.Println("Write error:", err)
					}
				}
			}()
		}
		dialog.Destroy()
	})

	// Open button (async)
	openBtnObj, _ := builder.GetObject("open_button")
	openBtn := openBtnObj.(*gtk.ToolButton)
	openBtn.Connect("clicked", func() {
		dialog, err := gtk.FileChooserDialogNewWith2Buttons(
			"Open File", window, gtk.FILE_CHOOSER_ACTION_OPEN,
			"Cancel", gtk.RESPONSE_CANCEL,
			"Open", gtk.RESPONSE_ACCEPT,
		)
		if err != nil {
			log.Fatal("Unable to create file chooser dialog:", err)
		}
		filter, _ := gtk.FileFilterNew()
		filter.AddPattern("*.txt")
		filter.AddPattern("*.rtf")
		dialog.AddFilter(filter)
		if dialog.Run() == gtk.RESPONSE_ACCEPT {
			filename := dialog.GetFilename()
			go func() {
				file, err := os.Open(filename)
				if err != nil {
					log.Println("Open error:", err)
					return
				}
				defer file.Close()
				data, err := io.ReadAll(file)
				if err != nil {
					log.Println("Read error:", err)
					return
				}
				glib.IdleAdd(func() bool {
					buffer.SetText(string(data))
					return false
				})
			}()
		}
		dialog.Destroy()
	})

	// Mail merge button
	mailMergeBtnObj, _ := builder.GetObject("mail_merge_button")
	mailMergeBtn := mailMergeBtnObj.(*gtk.ToolButton)
	mailMergeBtn.Connect("clicked", func() {
		mailMergeDialog(window, db)
	})

	// Manage data button
	manageDataBtnObj, _ := builder.GetObject("manage_data_button")
	manageDataBtn := manageDataBtnObj.(*gtk.ToolButton)
	manageDataBtn.Connect("clicked", func() {
		createTableDialog(window, db)
	})

	// Window close
	window.Connect("destroy", gtk.MainQuit)
	window.ShowAll()
	gtk.Main()
}

// Create table dialog
func createTableDialog(parent *gtk.Window, db *sql.DB) {
	dialog, err := gtk.DialogNew()
	if err != nil {
		log.Fatal("Unable to create dialog:", err)
	}
	dialog.SetTitle("Manage Table")
	dialog.SetTransientFor(parent)
	dialog.SetModal(true)
	dialog.AddButton("Create Table", gtk.RESPONSE_ACCEPT)
	dialog.AddButton("Edit Data", gtk.RESPONSE_APPLY)
	dialog.AddButton("Cancel", gtk.RESPONSE_CANCEL)

	vbox, err := dialog.GetContentArea()
	if err != nil {
		log.Fatal("Unable to get content area:", err)
	}

	// Table name
	nameLabel, err := gtk.LabelNew("Table Name:")
	if err != nil {
		log.Fatal("Unable to create label:", err)
	}
	nameEntry, err := gtk.EntryNew()
	if err != nil {
		log.Fatal("Unable to create entry:", err)
	}
	nameEntry.SetText("contacts")
	vbox.PackStart(nameLabel, false, false, 5)
	vbox.PackStart(nameEntry, false, false, 5)

	// Columns
	columns := make([]Column, 0, 15)
	listBox, err := gtk.ListBoxNew()
	if err != nil {
		log.Fatal("Unable to create list box:", err)
	}
	vbox.PackStart(listBox, true, true, 5)

	// Add column button
	addColumnBtn, err := gtk.ButtonNewWithLabel("Add Column")
	if err != nil {
		log.Fatal("Unable to create button:", err)
	}
	vbox.PackStart(addColumnBtn, false, false, 5)
	addColumnBtn.Connect("clicked", func() {
		if len(columns) >= 15 {
			messageDialog(parent, "Error", "Maximum 15 columns allowed")
			return
		}
		colDialog, err := gtk.DialogNew()
		if err != nil {
			log.Fatal("Unable to create column dialog:", err)
		}
		colDialog.SetTitle("Add Column")
		colDialog.SetTransientFor(dialog)
		colDialog.SetModal(true)
		colDialog.AddButton("OK", gtk.RESPONSE_ACCEPT)
		colDialog.AddButton("Cancel", gtk.RESPONSE_CANCEL)

		colVbox, err := colDialog.GetContentArea()
		if err != nil {
			log.Fatal("Unable to get column dialog content area:", err)
		}
		colNameLabel, err := gtk.LabelNew("Column Name:")
		if err != nil {
			log.Fatal("Unable to create label:", err)
		}
		colNameEntry, err := gtk.EntryNew()
		if err != nil {
			log.Fatal("Unable to create entry:", err)
		}
		colTypeLabel, err := gtk.LabelNew("Type:")
		if err != nil {
			log.Fatal("Unable to create label:", err)
		}
		colTypeCombo, err := gtk.ComboBoxTextNew()
		if err != nil {
			log.Fatal("Unable to create combo box:", err)
		}
		colTypeCombo.AppendText("VARCHAR")
		colTypeCombo.AppendText("DATE")
		colTypeCombo.SetActive(0)
		colVbox.PackStart(colNameLabel, false, false, 5)
		colVbox.PackStart(colNameEntry, false, false, 5)
		colVbox.PackStart(colTypeLabel, false, false, 5)
		colVbox.PackStart(colTypeCombo, false, false, 5)
		colVbox.ShowAll()
		if colDialog.Run() == gtk.RESPONSE_ACCEPT {
			name, err := colNameEntry.GetText()
			if err != nil {
				log.Fatal("Unable to get entry text:", err)
			}
			if name == "" {
				messageDialog(parent, "Error", "Column name cannot be empty")
			} else {
				columns = append(columns, Column{Name: name, Type: colTypeCombo.GetActiveText()})
				row, err := gtk.ListBoxRowNew()
				if err != nil {
					log.Fatal("Unable to create list box row:", err)
				}
				label, err := gtk.LabelNew(fmt.Sprintf("%s (%s)", name, colTypeCombo.GetActiveText()))
				if err != nil {
					log.Fatal("Unable to create label:", err)
				}
				row.Add(label)
				listBox.Add(row)
				listBox.ShowAll()
			}
		}
		colDialog.Destroy()
	})

	vbox.ShowAll()
	response := dialog.Run()
	if response == gtk.RESPONSE_ACCEPT {
		// Create table async
		tableName, err := nameEntry.GetText()
		if err != nil {
			log.Fatal("Unable to get entry text:", err)
		}
		if tableName == "" || len(columns) == 0 {
			messageDialog(parent, "Error", "Table name and at least one column required")
		} else {
			go createTable(db, tableName, columns)
			messageDialog(parent, "Success", "Table created")
		}
	} else if response == gtk.RESPONSE_APPLY {
		// Edit data
		tableName, err := nameEntry.GetText()
		if err != nil {
			log.Fatal("Unable to get entry text:", err)
		}
		if tableName == "" {
			messageDialog(parent, "Error", "Table name required")
		} else {
			editDataDialog(parent, db, tableName)
		}
	}
	dialog.Destroy()
}

// Create table in SQLite
func createTable(db *sql.DB, tableName string, columns []Column) {
	// Drop existing table
	_, err := db.Exec("DROP TABLE IF EXISTS " + tableName)
	if err != nil {
		log.Println("Drop table error:", err)
		return
	}

	// Create table
	var colDefs []string
	for _, col := range columns {
		colDefs = append(colDefs, fmt.Sprintf("%s %s", col.Name, col.Type))
	}
	query := fmt.Sprintf("CREATE TABLE %s (%s)", tableName, strings.Join(colDefs, ", "))
	_, err = db.Exec(query)
	if err != nil {
		log.Println("Create table error:", err)
	}
}

// Edit data dialog
func editDataDialog(parent *gtk.Window, db *sql.DB, tableName string) {
	dialog, err := gtk.DialogNew()
	if err != nil {
		log.Fatal("Unable to create dialog:", err)
	}
	dialog.SetTitle("Edit Data")
	dialog.SetTransientFor(parent)
	dialog.SetModal(true)
	dialog.AddButton("Add Row", gtk.RESPONSE_ACCEPT)
	dialog.AddButton("Done", gtk.RESPONSE_CANCEL)

	vbox, err := dialog.GetContentArea()
	if err != nil {
		log.Fatal("Unable to get content area:", err)
	}

	// Get columns
	rows, err := db.Query("PRAGMA table_info(" + tableName + ")")
	if err != nil {
		messageDialog(parent, "Error", "Failed to get table info")
		dialog.Destroy()
		return
	}
	var columns []Column
	for rows.Next() {
		var cid int
		var name, colType string
		var notnull, pk int
		var dflt_value *string
		err = rows.Scan(&cid, &name, &colType, &notnull, &dflt_value, &pk)
		if err != nil {
			log.Println("Scan error:", err)
			continue
		}
		columns = append(columns, Column{Name: name, Type: colType})
	}
	rows.Close()

	if len(columns) == 0 {
		messageDialog(parent, "Error", "Table has no columns")
		dialog.Destroy()
		return
	}

	// Grid for data
	treeView, err := gtk.TreeViewNew()
	if err != nil {
		log.Fatal("Unable to create tree view:", err)
	}
	types := make([]glib.Type, len(columns))
	for i := range types {
		types[i] = glib.TYPE_STRING
	}
	store, err := gtk.ListStoreNew(types...)
	if err != nil {
		log.Fatal("Unable to create list store:", err)
	}
	treeView.SetModel(store)
	for i, col := range columns {
		renderer, err := gtk.CellRendererTextNew()
		if err != nil {
			log.Fatal("Unable to create cell renderer:", err)
		}
		renderer.SetProperty("editable", true)
		renderer.Connect("edited", func(_ *gtk.CellRendererText, path, text string) {
			iter, err := store.GetIterFromString(path)
			if err != nil {
				log.Println("Get iter error:", err)
				return
			}
			store.SetValue(iter, i, text)
			// Update database async
			go updateRow(db, tableName, columns, store, iter)
		})
		column, err := gtk.TreeViewColumnNew()
		if err != nil {
			log.Fatal("Unable to create tree view column:", err)
		}
		column.SetTitle(col.Name)
		column.PackStart(renderer, true)
		column.AddAttribute(renderer, "text", i)
		treeView.AppendColumn(column)
	}
	scrolled, err := gtk.ScrolledWindowNew(nil, nil)
	if err != nil {
		log.Fatal("Unable to create scrolled window:", err)
	}
	scrolled.Add(treeView)
	vbox.PackStart(scrolled, true, true, 5)

	// Load existing data
	rows, err = db.Query("SELECT * FROM " + tableName)
	if err == nil {
		for rows.Next() {
			vals := make([]interface{}, len(columns))
			for i := range vals {
				vals[i] = new(string)
			}
			err = rows.Scan(vals...)
			if err != nil {
				log.Println("Scan error:", err)
				continue
			}
			iter := store.Append()
			for i, val := range vals {
				store.SetValue(iter, i, *(val.(*string)))
			}
		}
		rows.Close()
	}

	vbox.ShowAll()
	for dialog.Run() == gtk.RESPONSE_ACCEPT {
		// Add row
		iter := store.Append()
		for i := range columns {
			store.SetValue(iter, i, "")
		}
		// Insert into database async
		go insertRow(db, tableName, columns, store, iter)
	}
	dialog.Destroy()
}

// Insert row into database
func insertRow(db *sql.DB, tableName string, columns []Column, store *gtk.ListStore, iter *gtk.TreeIter) {
	var values []string
	for i := range columns {
		val, err := store.GetValue(iter, i)
		if err != nil {
			log.Println("Get value error:", err)
			return
		}
		str, err := val.GetString()
		if err != nil {
			log.Println("Get string error:", err)
			return
		}
		if columns[i].Type == "DATE" && str != "" {
			if !isValidDate(str) {
				log.Println("Invalid date format:", str)
				return
			}
		}
		values = append(values, fmt.Sprintf("'%s'", str))
	}
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tableName, strings.Join(colNames(columns), ", "), strings.Join(values, ", "))
	_, err := db.Exec(query)
	if err != nil {
		log.Println("Insert error:", err)
	}
}

// Update row in database
func updateRow(db *sql.DB, tableName string, columns []Column, store *gtk.ListStore, iter *gtk.TreeIter) {
	var sets []string
	for i, col := range columns {
		val, err := store.GetValue(iter, i)
		if err != nil {
			log.Println("Get value error:", err)
			return
		}
		str, err := val.GetString()
		if err != nil {
			log.Println("Get string error:", err)
			return
		}
		if col.Type == "DATE" && str != "" {
			if !isValidDate(str) {
				log.Println("Invalid date format:", str)
				return
			}
		}
		sets = append(sets, fmt.Sprintf("%s = '%s'", col.Name, str))
	}
	// Assume first column is unique for simplicity
	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s = '%s'",
		tableName, strings.Join(sets, ", "), columns[0].Name, getIterValue(store, iter, 0))
	_, err := db.Exec(query)
	if err != nil {
		log.Println("Update error:", err)
	}
}

// Mail merge function
func mailMerge(db *sql.DB, templateFile, outputFolder string) {
	// Read template async
	templateData, err := os.ReadFile(templateFile)
	if err != nil {
		log.Println("Template read error:", err)
		return
	}
	template := string(templateData)

	// Get columns
	rows, err := db.Query("PRAGMA table_info(contacts)")
	if err != nil {
		log.Println("Table info error:", err)
		return
	}
	var columns []string
	for rows.Next() {
		var cid int
		var name, colType string
		var notnull, pk int
		var dflt_value *string
		err = rows.Scan(&cid, &name, &colType, &notnull, &dflt_value, &pk)
		if err != nil {
			log.Println("Scan error:", err)
			continue
		}
		columns = append(columns, name)
	}
	rows.Close()

	// Query data
	query := fmt.Sprintf("SELECT %s FROM contacts", strings.Join(columns, ", "))
	rows, err = db.Query(query)
	if err != nil {
		log.Println("Query error:", err)
		return
	}
	defer rows.Close()

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 50) // Limit goroutines for Win98
	for rows.Next() {
		vals := make([]interface{}, len(columns))
		for i := range vals {
			vals[i] = new(string)
		}
		err = rows.Scan(vals...)
		if err != nil {
			log.Println("Scan error:", err)
			continue
		}
		wg.Add(1)
		semaphore <- struct{}{} // Acquire
		go func(vals []interface{}) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release
			content := template
			name := ""
			for i, val := range vals {
				str := *(val.(*string))
				content = strings.ReplaceAll(content, "{{"+columns[i]+"}}", str)
				if columns[i] == "Name" {
					name = str
				}
			}
			safeName := strings.ReplaceAll(strings.ToLower(name), " ", "_")
			outputFile := filepath.Join(outputFolder, "resume_"+safeName+".txt")
			file, err := os.Create(outputFile)
			if err != nil {
				log.Println("Create error:", err)
				return
			}
			defer file.Close()
			_, err = io.WriteString(file, content)
			if err != nil {
				log.Println("Write error:", err)
			}
		}(vals)
	}
	wg.Wait()
	log.Println("Mail merge complete")
}

// Mail merge dialog
func mailMergeDialog(parent *gtk.Window, db *sql.DB) {
	dialog, err := gtk.DialogNew()
	if err != nil {
		log.Fatal("Unable to create dialog:", err)
	}
	dialog.SetTitle("Mail Merge")
	dialog.SetTransientFor(parent)
	dialog.SetModal(true)
	dialog.AddButton("Run Merge", gtk.RESPONSE_ACCEPT)
	dialog.AddButton("Cancel", gtk.RESPONSE_CANCEL)

	dialog.SetDefaultSize(300, 200) // For Win98 640x480
	vbox, err := dialog.GetContentArea()
	if err != nil {
		log.Fatal("Unable to get content area:", err)
	}

	// Main vertical box
	box, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 5)
	if err != nil {
		log.Fatal("Unable to create box:", err)
	}
	vbox.PackStart(box, true, true, 5)

	// Grid for inputs
	grid, err := gtk.GridNew()
	if err != nil {
		log.Fatal("Unable to create grid:", err)
	}
	grid.SetRowSpacing(5)
	grid.SetColumnSpacing(5)
	box.PackStart(grid, false, false, 5)

	// Template file
	templateLabel, err := gtk.LabelNew("Template File:")
	if err != nil {
		log.Fatal("Unable to create label:", err)
	}
	templateEntry, err := gtk.EntryNew()
	if err != nil {
		log.Fatal("Unable to create entry:", err)
	}
	templateEntry.SetEditable(false)
	templateButton, err := gtk.ButtonNewWithLabel("Select")
	if err != nil {
		log.Fatal("Unable to create button:", err)
	}
	grid.Attach(templateLabel, 0, 0, 1, 1)
	grid.Attach(templateEntry, 1, 0, 1, 1)
	grid.Attach(templateButton, 2, 0, 1, 1)

	// Output folder
	outputLabel, err := gtk.LabelNew("Output Folder:")
	if err != nil {
		log.Fatal("Unable to create label:", err)
	}
	outputEntry, err := gtk.EntryNew()
	if err != nil {
		log.Fatal("Unable to create entry:", err)
	}
	outputEntry.SetEditable(false)
	outputButton, err := gtk.ButtonNewWithLabel("Select")
	if err != nil {
		log.Fatal("Unable to create button:", err)
	}
	grid.Attach(outputLabel, 0, 1, 1, 1)
	grid.Attach(outputEntry, 1, 1, 1, 1)
	grid.Attach(outputButton, 2, 1, 1, 1)

	// Data preview
	previewLabel, err := gtk.LabelNew("Data Preview:")
	if err != nil {
		log.Fatal("Unable to create label:", err)
	}
	box.PackStart(previewLabel, false, false, 5)
	treeView, err := gtk.TreeViewNew()
	if err != nil {
		log.Fatal("Unable to create tree view:", err)
	}
	store, err := gtk.ListStoreNew([]glib.Type{}...)
	if err != nil {
		log.Fatal("Unable to create list store:", err)
	}
	treeView.SetModel(store)
	scrolled, err := gtk.ScrolledWindowNew(nil, nil)
	if err != nil {
		log.Fatal("Unable to create scrolled window:", err)
	}
	scrolled.SetPolicy(gtk.POLICY_AUTOMATIC, gtk.POLICY_AUTOMATIC)
	scrolled.Add(treeView)
	scrolled.SetMinContentHeight(100)
	box.PackStart(scrolled, true, true, 5)

	// Load columns and data
	rows, err := db.Query("PRAGMA table_info(contacts)")
	if err == nil {
		var columns []string
		for rows.Next() {
			var cid int
			var name, colType string
			var notnull, pk int
			var dflt_value *string
			err = rows.Scan(&cid, &name, &colType, &notnull, &dflt_value, &pk)
			if err != nil {
				log.Println("Scan error:", err)
				continue
			}
			columns = append(columns, name)
			renderer, err := gtk.CellRendererTextNew()
			if err != nil {
				log.Fatal("Unable to create cell renderer:", err)
			}
			column, err := gtk.TreeViewColumnNew()
			if err != nil {
				log.Fatal("Unable to create tree view column:", err)
			}
			column.SetTitle(name)
			column.PackStart(renderer, true)
			column.AddAttribute(renderer, "text", len(columns)-1)
			treeView.AppendColumn(column)
		}
		rows.Close()
		types := make([]glib.Type, len(columns))
		for i := range types {
			types[i] = glib.TYPE_STRING
		}
		store, err = gtk.ListStoreNew(types...)
		if err != nil {
			log.Fatal("Unable to create list store:", err)
		}
		treeView.SetModel(store)
		rows, err = db.Query("SELECT * FROM contacts")
		if err == nil {
			for rows.Next() {
				vals := make([]interface{}, len(columns))
				for i := range vals {
					vals[i] = new(string)
				}
				err = rows.Scan(vals...)
				if err != nil {
					log.Println("Scan error:", err)
					continue
				}
				iter := store.Append()
				for i, val := range vals {
					store.SetValue(iter, i, *(val.(*string)))
				}
			}
			rows.Close()
		}
	}

	// Template button
	templateButton.Connect("clicked", func() {
		fileDialog, err := gtk.FileChooserDialogNewWith2Buttons(
			"Select Template", parent, gtk.FILE_CHOOSER_ACTION_OPEN,
			"Cancel", gtk.RESPONSE_CANCEL,
			"Select", gtk.RESPONSE_ACCEPT,
		)
		if err != nil {
			log.Fatal("Unable to create file chooser dialog:", err)
		}
		filter, err := gtk.FileFilterNew()
		if err != nil {
			log.Fatal("Unable to create file filter:", err)
		}
		filter.AddPattern("*.txt")
		fileDialog.AddFilter(filter)
		if fileDialog.Run() == gtk.RESPONSE_ACCEPT {
			filename := fileDialog.GetFilename()
			templateEntry.SetText(filename)
		}
		fileDialog.Destroy()
	})

	// Output button
	outputButton.Connect("clicked", func() {
		folderDialog, err := gtk.FileChooserDialogNewWith2Buttons(
			"Select Output Folder", parent, gtk.FILE_CHOOSER_ACTION_SELECT_FOLDER,
			"Cancel", gtk.RESPONSE_CANCEL,
			"Select", gtk.RESPONSE_ACCEPT,
		)
		if err != nil {
			log.Fatal("Unable to create folder chooser dialog:", err)
		}
		if folderDialog.Run() == gtk.RESPONSE_ACCEPT {
			filename := folderDialog.GetFilename()
			outputEntry.SetText(filename)
		}
		folderDialog.Destroy()
	})

	// Win98-style CSS
	cssProvider, err := gtk.CssProviderNew()
	if err != nil {
		log.Fatal("Unable to create CSS provider:", err)
	}
	err = cssProvider.LoadFromData("treeview { background-color: #C0C0C0; }")
	if err != nil {
		log.Fatal("Unable to load CSS data:", err)
	}
	screen, err := gdk.ScreenGetDefault()
	if err != nil {
		log.Fatal("Unable to get default screen:", err)
	}
	gtk.AddProviderForScreen(screen, cssProvider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)

	vbox.ShowAll()
	if dialog.Run() == gtk.RESPONSE_ACCEPT {
		templateFile, err := templateEntry.GetText()
		if err != nil {
			log.Fatal("Unable to get entry text:", err)
		}
		outputFolder, err := outputEntry.GetText()
		if err != nil {
			log.Fatal("Unable to get entry text:", err)
		}
		if templateFile == "" || outputFolder == "" {
			messageDialog(parent, "Error", "Template file and output folder required")
		} else {
			// Show progress dialog
			progressDialog := gtk.MessageDialogNew(parent, gtk.DIALOG_MODAL, gtk.MESSAGE_INFO, gtk.BUTTONS_NONE, "Merging...")
			go func() {
				mailMerge(db, templateFile, outputFolder)
				glib.IdleAdd(func() bool {
					progressDialog.Destroy()
					return false
				})
			}()
			progressDialog.Run()
		}
	}
	dialog.Destroy()
}

// Helper functions
func colNames(columns []Column) []string {
	names := make([]string, len(columns))
	for i, col := range columns {
		names[i] = col.Name
	}
	return names
}

func getIterValue(store *gtk.ListStore, iter *gtk.TreeIter, col int) string {
	val, err := store.GetValue(iter, col)
	if err != nil {
		log.Println("Get value error:", err)
		return ""
	}
	str, err := val.GetString()
	if err != nil {
		log.Println("Get string error:", err)
		return ""
	}
	return str
}

func isValidDate(date string) bool {
	_, err := time.Parse("2006-01-02", date)
	return err == nil
}

func messageDialog(parent *gtk.Window, title, message string) {
	dialog := gtk.MessageDialogNew(parent, gtk.DIALOG_MODAL, gtk.MESSAGE_INFO, gtk.BUTTONS_OK, message)
	dialog.SetTitle(title)
	dialog.Run()
	dialog.Destroy()
}
