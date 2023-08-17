package com.github.igrmk.smsq

import com.google.crypto.tink.CleartextKeysetHandle
import com.google.crypto.tink.JsonKeysetReader

object Constants {
    @Suppress("ConstantConditionIf")
    val BOT_NAME = if (BuildConfig.BUILD_TYPE == "staging") "DietarySupplements_VN_bot" else "DietarySupplements_VN_bot"
    const val LOG_HALVING_SIZE = 100000
    const val LOG_FILE_NAME = "log"
    const val DEFAULT_DOMAIN_NAME = "wasp-alive-basically.ngrok-free.app"
    const val PREFERENCES = "com.github.igrmk.smsq.preferences"
    const val PREF_DOMAIN_NAME = "domain_name"
    const val PREF_KEY = "key"
    const val PREF_ON = "on"
    const val PREF_CARRIER = "show_carrier"
    const val PREF_CONSENT = "consent"
    const val PREF_RETIRED = "retired"
    const val PREF_VERSION_CODE = "version_code"
    const val PREF_DELIVERED = "delivered"
    const val SOCKET_TIMEOUT_MS = 10000
    val RESEND_PERIOD_MS = arrayOf(5L * 60 * 1000, 15L * 60 * 1000, 45L * 60 * 1000)
    const val KEY_LENGTH = 64
    const val PERMISSIONS_SMS = 1
    const val PERMISSIONS_STATE = 2

    @Suppress("SpellCheckingInspection")
    const val RELEASE_PUBLIC_KEY_STRING = """
        {
  "primaryKeyId": 486731450,
  "key": [
    {
      "keyData": {
        "typeUrl": "type.googleapis.com/google.crypto.tink.EciesAeadHkdfPublicKey",
        "value": "ElwKBAgCEAMSUhJQCjh0eXBlLmdvb2dsZWFwaXMuY29tL2dvb2dsZS5jcnlwdG8udGluay5BZXNDdHJIbWFjQWVhZEtleRISCgYKAggQEBASCAoECAMQEBAgGAEYARogg8+t42l4OlrYJ11hx85GoMYhPuuvJXRoDWn8Srcag3wiIMKkd2vULG5BCuF7XEQhDomnGcC+l33/QrzIOi1R7MJ/",
        "keyMaterialType": "ASYMMETRIC_PUBLIC"
      },
      "status": "ENABLED",
      "keyId": 486731450,
      "outputPrefixType": "TINK"
    }
  ]
}
    """

    @Suppress("SpellCheckingInspection")
    const val STAGING_PUBLIC_KEY_STRING = """
        {
  "primaryKeyId": 486731450,
  "key": [
    {
      "keyData": {
        "typeUrl": "type.googleapis.com/google.crypto.tink.EciesAeadHkdfPublicKey",
        "value": "ElwKBAgCEAMSUhJQCjh0eXBlLmdvb2dsZWFwaXMuY29tL2dvb2dsZS5jcnlwdG8udGluay5BZXNDdHJIbWFjQWVhZEtleRISCgYKAggQEBASCAoECAMQEBAgGAEYARogg8+t42l4OlrYJ11hx85GoMYhPuuvJXRoDWn8Srcag3wiIMKkd2vULG5BCuF7XEQhDomnGcC+l33/QrzIOi1R7MJ/",
        "keyMaterialType": "ASYMMETRIC_PUBLIC"
      },
      "status": "ENABLED",
      "keyId": 486731450,
      "outputPrefixType": "TINK"
    }
  ]
}
    """

    @Suppress("ConstantConditionIf")
    val PUBLIC_KEY = CleartextKeysetHandle.read(JsonKeysetReader.withBytes(
            (if (BuildConfig.BUILD_TYPE == "staging") STAGING_PUBLIC_KEY_STRING else RELEASE_PUBLIC_KEY_STRING)
                    .toByteArray()))!!
}
