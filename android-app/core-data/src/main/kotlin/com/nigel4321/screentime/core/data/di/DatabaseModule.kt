package com.nigel4321.screentime.core.data.di

import android.content.Context
import androidx.room.Room
import com.nigel4321.screentime.core.data.cache.ScreentimeDatabase
import com.nigel4321.screentime.core.data.cache.UsageSummaryDao
import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.android.qualifiers.ApplicationContext
import dagger.hilt.components.SingletonComponent
import java.time.Clock
import javax.inject.Singleton

@Module
@InstallIn(SingletonComponent::class)
object DatabaseModule {
    @Provides
    @Singleton
    fun provideDatabase(
        @ApplicationContext context: Context,
    ): ScreentimeDatabase =
        Room.databaseBuilder(
            context,
            ScreentimeDatabase::class.java,
            ScreentimeDatabase.NAME,
        ).build()

    @Provides
    fun provideUsageSummaryDao(db: ScreentimeDatabase): UsageSummaryDao = db.usageSummaryDao()

    @Provides
    @Singleton
    fun provideClock(): Clock = Clock.systemUTC()
}
