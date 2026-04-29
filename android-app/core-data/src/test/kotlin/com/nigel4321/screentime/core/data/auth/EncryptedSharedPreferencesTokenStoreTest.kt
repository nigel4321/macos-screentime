package com.nigel4321.screentime.core.data.auth

import android.content.Context
import androidx.test.core.app.ApplicationProvider
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner

@RunWith(RobolectricTestRunner::class)
class EncryptedSharedPreferencesTokenStoreTest {
    private val context: Context = ApplicationProvider.getApplicationContext()

    @Test
    fun `set and current round-trip`() {
        val store = EncryptedSharedPreferencesTokenStore(context)

        store.set("jwt-abc")

        assertEquals("jwt-abc", store.current())
        assertEquals(AuthState.Authenticated("jwt-abc"), store.authState.value)
    }

    @Test
    fun `set null clears the persisted token`() {
        val store = EncryptedSharedPreferencesTokenStore(context)
        store.set("jwt-abc")

        store.set(null)

        assertNull(store.current())
        assertEquals(AuthState.Anonymous, store.authState.value)
    }

    @Test
    fun `token survives a fresh store instance backed by the same context`() {
        EncryptedSharedPreferencesTokenStore(context).set("persisted-jwt")

        val reopened = EncryptedSharedPreferencesTokenStore(context)

        assertEquals("persisted-jwt", reopened.current())
        assertEquals(AuthState.Authenticated("persisted-jwt"), reopened.authState.value)
    }
}
