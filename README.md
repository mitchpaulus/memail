# Mitch Email System

I'm pretty much over Outlook.

Here's the general outline that I'm thinking of:

- Emails as files, all stored in cloud blob storage (like Backblaze B2 to start), content addressed using BLAKE3 hash
- Local cache of files as needed
- local database (Sqlite?, maybe future is cloud shared database) that has only metadata:
  - BLAKE3 hash of the full email file.
  - From
  - To
  - Subject
  - Date
  - Has attachments
  - Tags (No folders)
  - Conversation/Message/Thread Id?
  - Read/Unread state?
  - Maybe text version of email?
- Simple TUI for filtering/searching/tagging emails
- Use browser for rendering HTML emails, otherwise use TUI for text emails
- Crossplatform
  - Windows CLI should work, shouldn't have to be through WSL.
