package com.github.igrmk.smsq.entities

import kotlinx.serialization.Polymorphic
import kotlinx.serialization.Serializable

@Serializable
@Polymorphic
open class Sms() {
    var id: Int = -1
    var type: String = TYPE_SMS
    var text: String = ""
    var sim: String = ""
    var carrier: String = ""
    var sender: String = ""
    var timestamp: Long = 0
    var offset: Int = 0

    constructor(sms: Sms) : this() {
        id = sms.id
        type = sms.type
        text = sms.text
        sim = sms.sim
        carrier = sms.carrier
        sender = sms.sender
        timestamp = sms.timestamp
        offset = sms.offset
    }

    companion object {
        const val TYPE_SMS = "sms"
        const val TYPE_INCOMING_CALL = "incoming_call"
    }
}
