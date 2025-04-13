# Productivity Tracker

A simple web application to track daily productivity.

## Features

- Record whether a day was productive or non-productive
- Data stored in SQLite database
- Simple web interface with notification system
- Automatic midnight check to add non-productive entries for days with no manual entry

## Requirements

- Go 1.23.7 or later
- SQLite

## Installation

1. Clone the repository:
   ```
   git clone https://github.com/ltratcliff/ambition.git
   cd ambition
   ```

2. Install dependencies:
   ```
   go mod tidy
   ```

   This will download all required dependencies, including the robfig/cron/v3 library used for scheduling the midnight check.

## Running the Application

1. Start the server:
   ```
   go run main.go
   ```

2. Open your browser and navigate to:
   ```
   http://localhost:3131/ambition
   ```

3. Click either the "Productive" or "Non-Productive" button to record your day's productivity.

## Database Schema

The application uses a SQLite database with the following schema:

```sql
CREATE TABLE productivity (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    date TEXT NOT NULL,
    productive INTEGER NOT NULL
);
```

- `id`: Unique identifier for each record
- `date`: Date of the productivity record (YYYY-MM-DD format)
- `productive`: 1 for productive, 0 for non-productive

## Automatic Midnight Check

The application includes a feature that automatically checks at midnight if an entry was made for the previous day. If no entry exists, it automatically adds a non-productive entry (productive=0) for that day.

This ensures that:
- Every day has an entry in the database
- Days without manual input are marked as non-productive by default

The feature works as follows:
1. When the server starts, it immediately checks if yesterday has an entry
2. It then schedules the next check for midnight
3. At midnight, it checks if the previous day has an entry and adds one if needed
4. This process repeats every day at midnight

### Cron-based Implementation

The application uses the robfig/cron library to schedule the midnight check. This provides a reliable and efficient way to schedule recurring tasks.

The cron scheduler is configured to run the check at midnight every day using the standard cron expression format: `0 0 * * *`

If you prefer to use the alternative goroutine-based implementation:

1. Comment out the cron-based implementation in main.go
2. Uncomment the goroutine-based implementation
3. Comment out the robfig/cron/v3 dependency in go.mod if desired

## Testing

To test the application:

1. Start the server
2. Open the web interface
3. Click on either button
4. Verify that a success notification appears
5. Check the database to confirm the entry was created:
   ```
   sqlite3 productivity.db "SELECT * FROM productivity;"
   ```
6. To test the midnight check feature, wait until midnight or manually change the system time

## License

[MIT License](LICENSE)
