package com.github.igrmk.smsq.helpers

import android.Manifest
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import android.telephony.SubscriptionInfo
import android.telephony.SubscriptionManager
import android.telephony.TelephonyManager
import androidx.core.app.ActivityCompat
import com.github.igrmk.smsq.entities.Sms
import com.github.igrmk.smsq.services.ResenderService
import java.util.*

class CallReceiver : BroadcastReceiver() {
    private val tag = this::class.simpleName!!

    private fun getDefaultSimInfo(context: Context): SubscriptionInfo? {
        if (ActivityCompat.checkSelfPermission(context, Manifest.permission.READ_PHONE_STATE) != PackageManager.PERMISSION_GRANTED) {
            return null
        }
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.LOLLIPOP_MR1) {
            val subscriptionManager = context.getSystemService(Context.TELEPHONY_SUBSCRIPTION_SERVICE) as SubscriptionManager
            val subs = subscriptionManager.activeSubscriptionInfoList
            return subs?.firstOrNull()
        }
        return null
    }

    override fun onReceive(context: Context, intent: Intent) {
        if (intent.action != TelephonyManager.ACTION_PHONE_STATE_CHANGED) {
            return
        }

        context.linf(tag, "action received: ${intent.action}, on: ${context.myPreferences.on}")
        if (!context.myPreferences.on) {
            return
        }

        val stateStr = intent.getStringExtra(TelephonyManager.EXTRA_STATE) ?: return
        val number = intent.getStringExtra(TelephonyManager.EXTRA_INCOMING_NUMBER)

        context.linf(tag, "phone state: $stateStr, number: $number")

        if (stateStr == TelephonyManager.EXTRA_STATE_RINGING && number != null) {
            context.linf(tag, "incoming call from: $number")
            processIncomingCall(context, number)
        }
    }

    private fun processIncomingCall(context: Context, number: String) {
        var displayName = ""
        var carrierName = ""

        if (context.myPreferences.showCarrier && Build.VERSION.SDK_INT >= Build.VERSION_CODES.LOLLIPOP_MR1) {
            val simInfo = getDefaultSimInfo(context)
            displayName = simInfo?.displayName?.toString() ?: ""
            carrierName = simInfo?.carrierName?.toString() ?: ""
        }

        val timestamp = System.currentTimeMillis()
        val cal = GregorianCalendar()
        val tz = cal.timeZone
        val offset = tz.getOffset(timestamp)

        val sms = Sms().apply {
            this.type = Sms.TYPE_INCOMING_CALL
            this.sim = displayName
            this.carrier = carrierName
            this.text = ""
            this.sender = number
            this.timestamp = timestamp / 1000
            this.offset = offset / 1000
        }

        context.storeSms(sms)
        context.startService(Intent(context, ResenderService::class.java))
    }
}
