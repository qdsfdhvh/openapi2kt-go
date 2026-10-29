package org.example.platform

import android.app.Activity
import android.app.Instrumentation
import android.os.Bundle

// Runs the common contracts without an Activity, permission or network request.
class RoadmapProbeInstrumentation : Instrumentation() {
    override fun onCreate(arguments: Bundle?) {
        super.onCreate(arguments)
        start()
    }

    override fun onStart() {
        val result = Bundle()
        try {
            main()
            result.putString("roadmap_result", "passed")
            result.putString("stream", "Platform model contracts passed, four Json settings")
            finish(Activity.RESULT_OK, result)
        } catch (failure: Throwable) {
            result.putString("roadmap_result", "failed")
            result.putString("stream", failure.javaClass.name + ": " + failure.message)
            finish(Activity.RESULT_CANCELED, result)
        }
    }
}
