#!/bin/sh
# PostgreSQL Backup Script
# Runs daily at 3 AM (configured in docker-compose)

BACKUP_DIR="/backup"
DATE=$(date +%Y%m%d_%H%M%S)
FILENAME="kita_backup_${DATE}.sql.gz"
KEEP_DAYS=30

echo "Starting backup: ${FILENAME}"

# Create backup
pg_dump | gzip > "${BACKUP_DIR}/${FILENAME}"

if [ $? -eq 0 ]; then
    echo "Backup completed successfully: ${FILENAME}"
    
    # Delete old backups
    find ${BACKUP_DIR} -name "kita_backup_*.sql.gz" -mtime +${KEEP_DAYS} -delete
    echo "Deleted backups older than ${KEEP_DAYS} days"
else
    echo "Backup failed!"
    exit 1
fi
