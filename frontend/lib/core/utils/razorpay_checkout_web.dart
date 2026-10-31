// ignore_for_file: avoid_web_libraries_in_flutter, deprecated_member_use
import 'dart:js' as js;

void openRazorpayCheckout({
  required String key,
  required int amount,
  required String currency,
  required String name,
  required String description,
  required String orderId,
  required String prefillName,
  required String prefillEmail,
  required void Function(String paymentId, String orderId, String signature)
      onSuccess,
  required void Function(String error) onFailure,
}) {
  try {
    final opts = js.JsObject.jsify({
      'key': key,
      'amount': amount,
      'currency': currency,
      'name': name,
      'description': description,
      'order_id': orderId,
      'prefill': {
        'name': prefillName,
        'email': prefillEmail,
      },
      'theme': {
        'color': '#6366f1',
      },
    });

    void callback(dynamic res) {
      if (res is Map || res is js.JsObject) {
        final status = res['status']?.toString();
        if (status == 'success') {
          final paymentId = res['payment_id']?.toString() ?? '';
          final resOrderId = res['order_id']?.toString() ?? orderId;
          final signature = res['signature']?.toString() ?? '';
          onSuccess(paymentId, resOrderId, signature);
          return;
        }
      }
      onFailure('Payment cancelled by user');
    }

    js.context.callMethod('openRazorpayModal', [opts, callback]);
  } catch (e) {
    onFailure('Failed to open Razorpay: $e');
  }
}
