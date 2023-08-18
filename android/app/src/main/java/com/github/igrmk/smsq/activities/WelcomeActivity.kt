package com.github.igrmk.smsq.activities

import android.Manifest
import android.content.*
import android.net.Uri
import android.os.Build
import android.os.Bundle
import android.os.Handler
import android.view.View
import androidx.appcompat.app.AlertDialog
import androidx.appcompat.app.AppCompatActivity
import androidx.core.app.ActivityCompat
import androidx.core.content.ContextCompat
import androidx.core.content.PermissionChecker.PERMISSION_GRANTED
import com.github.igrmk.smsq.Constants
import com.github.igrmk.smsq.R
import com.github.igrmk.smsq.helpers.*
import com.github.igrmk.smsq.services.ResenderService
import kotlinx.android.synthetic.main.activity_welcome.*
import android.provider.Telephony.Sms.Intents.SMS_RECEIVED_ACTION
import androidx.localbroadcastmanager.content.LocalBroadcastManager

class WelcomeActivity : AppCompatActivity() {
    private val tag = this::class.simpleName!!
    private val versionClickHandler = Handler()
    private var versionClicks = 0
    private val decreaseVersionClicks = Runnable { versionClicks-- }

    private val deliveredListener = object : BroadcastReceiver() {
        override fun onReceive(context: Context, intent: Intent) {
            delivered.text = myPreferences.delivered.toString()
        }
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_welcome)
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.LOLLIPOP_MR1) {
            showCarrier.isEnabled = false
        }
    }

    override fun onResume() {
        linf(tag, "resuming activity...")
        super.onResume()
        resume()
        val lbm = LocalBroadcastManager.getInstance(this)
        lbm.registerReceiver(deliveredListener, IntentFilter(Constants.PREF_DELIVERED))
    }

    override fun onPause() {
        super.onPause()
        versionClickHandler.removeCallbacks(decreaseVersionClicks)
        val lbm = LocalBroadcastManager.getInstance(this)
        lbm.unregisterReceiver(deliveredListener)
    }

    private fun resume() {
        versionClicks = 0
        delivered.text = myPreferences.delivered.toString()
        if (myPreferences.retired) {
            retired()
            return
        }
        logSmsReceivers()
        if (!myPreferences.consent) {
            consentAlert()
            return
        }
        if (myPreferences.on) {
            checkStart()
        }
        if (myPreferences.showCarrier) {
            checkShowCarrierSwitch()
        }

        botIdText.setText(myPreferences.botName);
        urlText.setText(myPreferences.domainName);
        botIdText.isEnabled = !myPreferences.on;
        urlText.isEnabled = !myPreferences.on;
    }

    private fun logSmsReceivers() {
        val intent = Intent(SMS_RECEIVED_ACTION)
        val infos = packageManager.queryBroadcastReceivers(intent, 0)
        linf(tag, "SMS receivers:")
        for (info in infos) {
            linf(tag, "name: ${info.activityInfo.name}, priority: ${info.priority}")
        }
    }

    private fun checkStart(): Boolean {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.M) {
            allowed()
            return true
        }
        if (ContextCompat.checkSelfPermission(this, Manifest.permission.RECEIVE_SMS) == PERMISSION_GRANTED) {
            allowed()
            return true
        }
        stop()
        return false
    }

    private fun start() {
        if (checkStart()) {
            return
        }
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M) {
            ActivityCompat.requestPermissions(this, arrayOf(Manifest.permission.RECEIVE_SMS), Constants.PERMISSIONS_SMS)
        }
    }

    private fun allowed() {
        myPreferences.on = true
        imageButton.setImageResource(R.drawable.ic_on)
        connect.isEnabled = true
        revokeButton.isEnabled = true
        start.text = getString(R.string.pause)
        if (myPreferences.key == null) {
            updateKey()
        }
        this.startService(Intent(this, ResenderService::class.java))
    }

    private fun stop() {
        myPreferences.on = false
        imageButton.setImageResource(R.drawable.ic_off)
        connect.isEnabled = false
        start.text = getString(R.string.start)
        this.stopService(Intent(this, ResenderService::class.java))
    }

    fun onStartClicked(@Suppress("UNUSED_PARAMETER") view: View) {
        if (!myPreferences.on) {

            val bot = botIdText.text.toString();
            if (bot != null && bot.trim().isNotEmpty()) {
                myPreferences.botName = bot;
            }


            val url = urlText.text.toString();
            if (url != null && url.trim().isNotEmpty()) {
                myPreferences.domainName = url;
            }

            botIdText.isEnabled = false;
            urlText.isEnabled = false;

            start()
        } else {
            stop()
            botIdText.isEnabled = true;
            urlText.isEnabled = true;
        }
    }

    fun onRevokeClicked(@Suppress("UNUSED_PARAMETER") view: View) {
        AlertDialog.Builder(this)
                .setMessage("Are you sure you want to revoke access to connected account?")
                .setPositiveButton(android.R.string.yes) { _, _ -> forceRevoke() }
                .setNegativeButton(android.R.string.cancel, null)
                .show()
    }

    private fun forceRevoke() {
        revokeKey()
        revokeButton.isEnabled = false
        stop()
    }

    private fun copyToClipboard(link: String) {
        val clipboard = getSystemService(CLIPBOARD_SERVICE) as ClipboardManager
        val clip = ClipData.newPlainText("smsQ connection link", link)
        clipboard.primaryClip = clip
    }

    private fun telegramAlert() {
        AlertDialog.Builder(this)
                .setMessage("Please install Telegram to connect the bot")
                .setPositiveButton(android.R.string.ok, null)
                .show()
    }

    private fun consentAlert() {
        AlertDialog.Builder(this)
                .setMessage("Allow this application to read new SMS messages including their text, sender information, time and carrier name?")
                .setPositiveButton(android.R.string.yes) { _, _ -> myPreferences.consent = true; resume() }
                .setNegativeButton(android.R.string.no) { _, _ -> finish() }
                .show()
    }

    private fun retired() {
        AlertDialog.Builder(this)
                .setMessage("Current app version is out of date. Please upgrade")
                .setPositiveButton(android.R.string.ok) { _, _ -> finish() }
                .show()
    }

    fun onConnectClicked(@Suppress("UNUSED_PARAMETER") view: View) {
        val link = "tg://resolve?domain=${myPreferences.botName}&start=${myPreferences.key}"
        try {
            startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(link)))
        } catch (ex: android.content.ActivityNotFoundException) {
            if (isProbablyAnEmulator()) {
                copyToClipboard(link)
                return
            }
            telegramAlert()
        }
    }

    private fun checkShowCarrierSwitch(): Boolean {
        if (myPreferences.showCarrier) {
            showCarrier.isChecked = true
        }
        if (ContextCompat.checkSelfPermission(this, Manifest.permission.READ_PHONE_STATE) == PERMISSION_GRANTED) {
            myPreferences.showCarrier = true
            return true
        }
        myPreferences.showCarrier = false
        showCarrier.isChecked = false
        return false
    }

    fun onCarrierClicked(@Suppress("UNUSED_PARAMETER") view: View) {
        if (!showCarrier.isChecked) {
            myPreferences.showCarrier = false
            return
        }
        if (checkShowCarrierSwitch()) {
            return
        }
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M) {
            ActivityCompat.requestPermissions(this, arrayOf(Manifest.permission.READ_PHONE_STATE), Constants.PERMISSIONS_STATE)
        }
    }

    override fun onRequestPermissionsResult(requestCode: Int, permissions: Array<String>, grantResults: IntArray) {
        when (requestCode) {
            Constants.PERMISSIONS_SMS ->
                if (grantResults.isNotEmpty() && grantResults.all { it == PERMISSION_GRANTED }) {
                    allowed()
                } else {
                    stop()
                }

            Constants.PERMISSIONS_STATE ->
                if (grantResults.isNotEmpty() && grantResults.all { it == PERMISSION_GRANTED }) {
                    showCarrier.isChecked = true
                    myPreferences.showCarrier = true
                } else {
                    showCarrier.isChecked = false
                    myPreferences.showCarrier = false
                }
        }
    }

    fun onPrivacyClick(@Suppress("UNUSED_PARAMETER") view: View) {
        startActivity(Intent(this, PrivacyActivity::class.java))
    }

    fun onVersionClick(@Suppress("UNUSED_PARAMETER") view: View) {
        versionClicks++
        versionClickHandler.postDelayed(decreaseVersionClicks, 2000)
        if (versionClicks == 5) {
            startActivity(Intent(this, DebugActivity::class.java))
        }
    }
}
