package com.nigel4321.screentime.core.data.cache

import androidx.room.Database
import androidx.room.RoomDatabase

@Database(
    entities = [UsageSummaryRowEntity::class],
    version = 1,
    exportSchema = true,
)
abstract class ScreentimeDatabase : RoomDatabase() {
    abstract fun usageSummaryDao(): UsageSummaryDao

    companion object {
        const val NAME: String = "screentime.db"
    }
}
