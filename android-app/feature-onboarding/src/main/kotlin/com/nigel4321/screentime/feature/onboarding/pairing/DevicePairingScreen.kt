package com.nigel4321.screentime.feature.onboarding.pairing

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.selection.selectable
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.RadioButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.nigel4321.screentime.core.domain.model.Device
import com.nigel4321.screentime.core.domain.model.DeviceId
import com.nigel4321.screentime.core.domain.model.DevicePlatform

@Composable
fun DevicePairingScreen(
    modifier: Modifier = Modifier,
    viewModel: DevicePairingViewModel = hiltViewModel(),
) {
    val state by viewModel.uiState.collectAsStateWithLifecycle()

    Column(
        modifier =
            modifier
                .fillMaxSize()
                .padding(24.dp),
    ) {
        Text(
            text = "Choose your primary device",
            style = MaterialTheme.typography.headlineMedium,
        )
        Spacer(Modifier.height(8.dp))
        Text(
            text = "We'll show usage from this device on your dashboards.",
            style = MaterialTheme.typography.bodyMedium,
        )
        Spacer(Modifier.height(24.dp))

        when (val current = state) {
            DevicePairingUiState.Loading ->
                Column(
                    modifier = Modifier.fillMaxSize(),
                    horizontalAlignment = Alignment.CenterHorizontally,
                    verticalArrangement = Arrangement.Center,
                ) {
                    CircularProgressIndicator()
                }

            is DevicePairingUiState.Devices ->
                DeviceList(
                    devices = current.devices,
                    onContinue = viewModel::selectAndContinue,
                )

            DevicePairingUiState.ZeroDevices -> ZeroDevicesState()

            is DevicePairingUiState.Error ->
                ErrorState(message = current.message, onRetry = viewModel::retry)
        }
    }
}

@Composable
private fun DeviceList(
    devices: List<Device>,
    onContinue: (DeviceId) -> Unit,
) {
    var selected by rememberSaveable(devices) { mutableStateOf(devices.firstOrNull()?.id?.value) }

    LazyColumn(modifier = Modifier.fillMaxWidth()) {
        items(devices, key = { it.id.value }) { device ->
            val isSelected = selected == device.id.value
            Row(
                modifier =
                    Modifier
                        .fillMaxWidth()
                        .selectable(
                            selected = isSelected,
                            onClick = { selected = device.id.value },
                        )
                        .padding(vertical = 12.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                RadioButton(selected = isSelected, onClick = null)
                Spacer(Modifier.height(0.dp))
                Column(modifier = Modifier.padding(start = 16.dp)) {
                    Text(
                        text = device.platform.displayName(),
                        style = MaterialTheme.typography.titleMedium,
                    )
                    Text(
                        text = device.fingerprint,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
        }
    }

    Spacer(Modifier.height(16.dp))
    Button(
        modifier = Modifier.fillMaxWidth(),
        enabled = selected != null,
        onClick = {
            val choice = selected
            if (choice != null) onContinue(DeviceId(choice))
        },
    ) {
        Text("Continue")
    }
}

@Composable
private fun ZeroDevicesState() {
    Column(
        modifier = Modifier.fillMaxSize(),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Text(
            text = "No devices yet",
            style = MaterialTheme.typography.titleLarge,
        )
        Spacer(Modifier.height(8.dp))
        Text(
            text = "Install the Mac agent first, then come back here.",
            style = MaterialTheme.typography.bodyMedium,
        )
    }
}

@Composable
private fun ErrorState(
    message: String,
    onRetry: () -> Unit,
) {
    Column(
        modifier = Modifier.fillMaxSize(),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Text(
            text = message,
            color = MaterialTheme.colorScheme.error,
            style = MaterialTheme.typography.bodyMedium,
        )
        Spacer(Modifier.height(12.dp))
        TextButton(onClick = onRetry) {
            Text("Retry")
        }
    }
}

private fun DevicePlatform.displayName(): String =
    when (this) {
        DevicePlatform.Macos -> "Mac"
        DevicePlatform.Android -> "Android"
        DevicePlatform.Unknown -> "Other device"
    }
