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
  onFailure('Razorpay checkout is only supported on web');
}
