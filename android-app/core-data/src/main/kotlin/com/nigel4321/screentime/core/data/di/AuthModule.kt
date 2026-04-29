package com.nigel4321.screentime.core.data.di

import com.nigel4321.screentime.core.data.auth.InMemoryTokenStore
import com.nigel4321.screentime.core.data.auth.TokenStore
import dagger.Binds
import dagger.Module
import dagger.hilt.InstallIn
import dagger.hilt.components.SingletonComponent
import javax.inject.Singleton

@Module
@InstallIn(SingletonComponent::class)
abstract class AuthModule {
    @Binds
    @Singleton
    abstract fun bindTokenStore(impl: InMemoryTokenStore): TokenStore
}
