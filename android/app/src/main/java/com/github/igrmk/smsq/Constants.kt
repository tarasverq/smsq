package com.github.igrmk.smsq

import com.google.crypto.tink.CleartextKeysetHandle
import com.google.crypto.tink.JsonKeysetReader

object Constants {
    @Suppress("ConstantConditionIf")
    val BOT_NAME = if (BuildConfig.BUILD_TYPE == "staging") "smsq_test_bot" else "smsq_bot"
    const val LOG_HALVING_SIZE = 100000
    const val LOG_FILE_NAME = "log"
    val DEFAULT_DOMAIN_NAME = if (BuildConfig.BUILD_TYPE == "staging") "https://apitest.smsq.me/" else "https://api.smsq.me/";
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
          "key": [{
              "keyData": {
                "typeUrl": "type.googleapis.com/google.crypto.tink.EciesAeadHkdfPublicKey",
                "keyMaterialType": "ASYMMETRIC_PUBLIC",
                "value": "ElwKBAgCEAMSUhJQCjh0eXBlLmdvb2dsZWFwaXMuY29tL2dvb2dsZS5jcnlwdG8udGluay5BZXNDdHJIbWFjQWVhZEtleRISCgYKAggQEBASCAoECAMQEBAgGAEYARogg8+t42l4OlrYJ11hx85GoMYhPuuvJXRoDWn8Srcag3wiIMKkd2vULG5BCuF7XEQhDomnGcC+l33/QrzIOi1R7MJ/"
              },
              "outputPrefixType": "TINK",
              "keyId": 486731450,
              "status": "ENABLED"
            }]
        }
    """

    @Suppress("SpellCheckingInspection")
    const val STAGING_PUBLIC_KEY_STRING = """
        {
            "primaryKeyId": 3412351950,
            "key": [{
                "keyData": {
                    "typeUrl": "type.googleapis.com/google.crypto.tink.EciesAeadHkdfPublicKey",
                    "keyMaterialType": "ASYMMETRIC_PUBLIC",
                    "value": "ElwKBAgCEAMSUhJQCjh0eXBlLmdvb2dsZWFwaXMuY29tL2dvb2dsZS5jcnlwdG8udGluay5BZXNDdHJIbWFjQWVhZEtleRISCgYKAggQEBASCAoECAMQEBAgGAEYARogGIRjU1iGu1eQ86LMS+BQRtccWYGMbh1FVEplotrBgxsiIEovOn1zuHshy3/EciMYwUmh5Rw6wRjSxpCaTlTSnWLU"
                },
                "outputPrefixType": "TINK",
                "keyId": 3412351950,
                "status": "ENABLED"
            }]
        }
    """

    @Suppress("ConstantConditionIf")
    val PUBLIC_KEY = CleartextKeysetHandle.read(JsonKeysetReader.withBytes(
            (if (BuildConfig.BUILD_TYPE == "staging") STAGING_PUBLIC_KEY_STRING else RELEASE_PUBLIC_KEY_STRING)
                    .toByteArray()))!!
}
