package com.nigel4321.screentime.feature.onboarding.di

import com.nigel4321.screentime.feature.onboarding.auth.CredentialManagerGoogleSignInClient
import com.nigel4321.screentime.feature.onboarding.auth.GoogleSignInClient
import dagger.Binds
import dagger.Module
import dagger.hilt.InstallIn
import dagger.hilt.components.SingletonComponent
import javax.inject.Singleton

@Module
@InstallIn(SingletonComponent::class)
abstract class OnboardingModule {
    @Binds
    @Singleton
    abstract fun bindGoogleSignInClient(impl: CredentialManagerGoogleSignInClient): GoogleSignInClient
}
