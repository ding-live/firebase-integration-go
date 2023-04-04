# Firebase with Ding OTP
This Go REST API shows how to authenticate users using the Firebase SDK while using Ding as an OTP provider.


## Setup
To get started, you need to setup a Firebase project in the Firebase console. Learn more [here](https://firebase.google.com/docs/guides).
### Environment
You need the following envs to connect to your Firebase project and the Ding API.
- `SA_FILE_PATH`: A path to your service account JSON file
- `DING_API_KEY`: Your Ding API key
- `DING_CUSTOMER_UUID`: Your Ding customer UUID