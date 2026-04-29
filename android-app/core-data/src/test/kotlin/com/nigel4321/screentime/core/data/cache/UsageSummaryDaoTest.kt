package com.nigel4321.screentime.core.data.cache

import androidx.room.Room
import androidx.test.core.app.ApplicationProvider
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.test.runTest
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner

@RunWith(RobolectricTestRunner::class)
class UsageSummaryDaoTest {
    private lateinit var db: ScreentimeDatabase
    private lateinit var dao: UsageSummaryDao

    @Before
    fun setUp() {
        db =
            Room.inMemoryDatabaseBuilder(
                ApplicationProvider.getApplicationContext(),
                ScreentimeDatabase::class.java,
            ).allowMainThreadQueries().build()
        dao = db.usageSummaryDao()
    }

    @After
    fun tearDown() {
        db.close()
    }

    @Test
    fun `insert and observe round-trip`() =
        runTest {
            dao.insertAll(
                listOf(
                    row("k1", "com.a", null, 60, 1_000L),
                    row("k1", null, "2026-04-29", 120, 1_000L),
                    row("k2", "com.b", null, 30, 1_000L),
                ),
            )

            val k1 = dao.observeByCacheKey("k1").first()

            assertEquals(2, k1.size)
            assertEquals("com.a", k1[0].bundleId)
            assertEquals("2026-04-29", k1[1].day)
        }

    @Test
    fun `lastRefreshAt returns the value written by upsertMetadata`() =
        runTest {
            dao.upsertMetadata(CacheMetadataEntity(cacheKey = "k1", lastRefreshAt = 1_500L))
            dao.upsertMetadata(CacheMetadataEntity(cacheKey = "k1", lastRefreshAt = 2_000L))

            assertEquals(2_000L, dao.lastRefreshAt("k1"))
        }

    @Test
    fun `lastRefreshAt returns null for an unknown key`() =
        runTest {
            assertNull(dao.lastRefreshAt("missing"))
        }

    @Test
    fun `replace deletes prior rows and writes the metadata in one transaction`() =
        runTest {
            dao.insertAll(listOf(row("k1", "old", null, 60, 1_000L)))

            dao.replace(
                cacheKey = "k1",
                rows = listOf(row("k1", "new", null, 60, 2_000L)),
                refreshedAt = 2_000L,
            )

            val rows = dao.observeByCacheKey("k1").first()
            assertEquals(1, rows.size)
            assertEquals("new", rows[0].bundleId)
            assertEquals(2_000L, dao.lastRefreshAt("k1"))
        }

    @Test
    fun `replace with empty rows still records freshness via metadata`() =
        runTest {
            dao.replace(cacheKey = "k1", rows = emptyList(), refreshedAt = 5_000L)

            assertEquals(0, dao.observeByCacheKey("k1").first().size)
            assertEquals(5_000L, dao.lastRefreshAt("k1"))
        }

    @Test
    fun `deleteOlderThan returns the count and prunes old rows across keys`() =
        runTest {
            dao.insertAll(
                listOf(
                    row("k1", "old1", null, 60, 1_000L),
                    row("k2", "old2", null, 60, 1_500L),
                    row("k1", "fresh", null, 60, 3_000L),
                ),
            )

            val deleted = dao.deleteOlderThan(2_000L)

            assertEquals(2, deleted)
            val k1 = dao.observeByCacheKey("k1").first()
            assertEquals(1, k1.size)
            assertEquals("fresh", k1[0].bundleId)
            assertEquals(0, dao.observeByCacheKey("k2").first().size)
        }

    private fun row(
        cacheKey: String,
        bundleId: String?,
        day: String?,
        durationSeconds: Long,
        cachedAt: Long,
    ): UsageSummaryRowEntity =
        UsageSummaryRowEntity(
            cacheKey = cacheKey,
            bundleId = bundleId,
            day = day,
            durationSeconds = durationSeconds,
            cachedAt = cachedAt,
        )
}
