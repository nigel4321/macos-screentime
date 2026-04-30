package com.nigel4321.screentime.core.data.cache

import androidx.room.Database
import androidx.room.RoomDatabase
import androidx.room.migration.Migration
import androidx.sqlite.db.SupportSQLiteDatabase

@Database(
    entities = [UsageSummaryRowEntity::class, CacheMetadataEntity::class],
    version = 2,
    exportSchema = true,
)
abstract class ScreentimeDatabase : RoomDatabase() {
    abstract fun usageSummaryDao(): UsageSummaryDao

    companion object {
        const val NAME: String = "screentime.db"

        /**
         * v1 → v2: add nullable `display_name` to `usage_summary_row`
         * for the §2.22 server-supplied app names. Default null on
         * existing rows so the next refresh repopulates from
         * `GET /v1/usage:summary`.
         */
        val MIGRATION_1_2: Migration =
            object : Migration(1, 2) {
                override fun migrate(db: SupportSQLiteDatabase) {
                    db.execSQL("ALTER TABLE usage_summary_row ADD COLUMN display_name TEXT")
                }
            }
    }
}
