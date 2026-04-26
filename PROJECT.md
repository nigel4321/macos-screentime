# Android app for MacOS screentime

## Objective
Build a native MacOS application that can read and set screen time restrictions for all application running on the device.
We will have an android application with a dashboard which allows us to view usage and manage screen time restrictions.
The Mac signs in with Apple and the Android app signs in with Google; the two devices pair to a single backend account via a one-time code so usage and policy are shared.

## Ways of working
- Use Swift for the MacOS application.
- Use Kotlin for the Android application.
- Use a test-driven development approach.
- Use GitHub workflows and actions to run tests.
  - Tests should run on pushes to branchs and on creating pull requests.
- Use an interative development approach. We will have a unit of testable functionality before moving on to the next
- Use GitHub workflows and actions with Fastlane to deploy to Google's play store internal testing track.
