GoATPAD
GoATPAD is a lightweight, open-source rich text editor and SQLite database manager with mail merge capabilities. Built with Go, GTK 3, and Glade, it blends a retro 90s word processor vibe with modern functionality. It’s perfect for creating formatted documents, managing data, and automating tasks—whether you’re on Linux, Windows, or macOS.

GoATPAD is licensed under the .

Features
Rich Text Editing: Bold, italic, font sizes (10–16 pt), and text alignment.
File Support: Save/open files as plain text (.txt) or basic RTF (.rtf).
SQLite Management: Create and edit tables (up to 15 columns) and manage data in-app.
Mail Merge: Generate personalized documents from SQLite data using templates.
CLI Batch Mode: Automate mail merges from the command line.
Building and Running
Prerequisites
Go 1.22.2+
GTK 3.24.41+ (with development libraries)
Glade 3.40.0+ (optional, for UI editing)
SQLite (bundled via modernc.org/sqlite)
On Ubuntu/Debian, install dependencies:

bash

Copy
sudo apt-get install libgtk-3-dev libglib2.0-dev libpango1.0-dev glade
Build Instructions
Clone the repo:
bash

Copy
git clone https://github.com/yourusername/goatpad.git
cd goatpad
Build the binary:
bash

Copy
go build -v -o goatpad
Run it:
bash

Copy
./goatpad
CLI Batch Mode
Run a mail merge without the GUI:

bash

Copy
./goatpad --batch-merge --template=template.txt --db=contacts.db --output=./output
Usage
Edit Text: Use the toolbar for formatting (bold, italic, etc.).
Manage Data: Click "Manage Data" to work with SQLite tables.
Mail Merge: Select "Mail Merge" to create documents from your data.
Contributing
We’d love your help! To contribute:

Fork the repository.
Create a branch (git checkout -b feature/your-feature).
Commit your changes (git commit -m "Add your feature").
Push to your branch (git push origin feature/your-feature).
Open a pull request.
License
GoATPAD is licensed under the . See the  file for details.