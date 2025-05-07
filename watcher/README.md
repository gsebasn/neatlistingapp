# MongoDB to Typesense Watcher

This service watches a MongoDB collection for changes and automatically syncs them to a Typesense search index.

## Features

- Real-time monitoring of MongoDB changes (insert, update, delete)
- Automatic synchronization with Typesense
- Configurable through environment variables
- Error handling and logging

## Prerequisites

- Go 1.21 or later
- MongoDB instance with change streams enabled
- Typesense instance

## Configuration

Create a `.env` file in the root directory with the following variables:

```env
MONGODB_URI=mongodb://localhost:27017
MONGODB_DATABASE=your_database
MONGODB_COLLECTION=your_collection

TYPESENSE_API_KEY=your_api_key
TYPESENSE_HOST=localhost
TYPESENSE_PORT=8108
TYPESENSE_PROTOCOL=http
TYPESENSE_COLLECTION_NAME=your_collection
```

## Installation

1. Clone the repository
2. Install dependencies:
   ```bash
   go mod download
   ```
3. Build the application:
   ```bash
   go build
   ```

## Usage

Run the application:

```bash
./watcher
```

The service will:
1. Connect to MongoDB and Typesense
2. Start watching the specified collection for changes
3. Automatically sync any changes to Typesense
4. Log all operations and any errors that occur

## Error Handling

The service includes comprehensive error handling:
- Connection errors are logged and the service will exit
- Document processing errors are logged but won't stop the service
- All operations are logged for monitoring

## Notes

- Make sure your MongoDB instance has change streams enabled
- The Typesense collection must exist before running the service
- The service assumes the document structure in MongoDB matches the schema in Typesense 