package com.nigel4321.screentime.core.data.di

import com.nigel4321.screentime.core.data.device.SelectedDeviceStore
import com.nigel4321.screentime.core.data.device.SharedPreferencesSelectedDeviceStore
import dagger.Binds
import dagger.Module
import dagger.hilt.InstallIn
import dagger.hilt.components.SingletonComponent
import javax.inject.Singleton

@Module
@InstallIn(SingletonComponent::class)
abstract class DeviceModule {
    @Binds
    @Singleton
    abstract fun bindSelectedDeviceStore(impl: SharedPreferencesSelectedDeviceStore): SelectedDeviceStore
}
