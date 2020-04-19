(function(){
  // Returns if browser supports the crypto api
  function supportsCrypto () {
    return window.crypto && crypto.subtle && window.TextEncoder;
  }

  // URL-safe base64 encode
  const encode64 = (buf) => {
    return btoa(new Uint8Array(buf).reduce((s, b) => s + String.fromCharCode(b), ''))
      .replace(/\+/g, '-') // Convert '+' to '-'
      .replace(/\//g, '_'); // Convert '/' to '_';
  }

  if (supportsCrypto()) {
    document.body.classList.add("crypto");
  }

  // Generate a _really_ random key
  document.querySelector(".generateKey").addEventListener("click", (event) => {
    event.preventDefault();

    const values = encode64(crypto.getRandomValues(new Uint8Array(12)));
    const field = document.querySelector("input[name='auth_key']");
    field.value = values;
  });

  document.querySelectorAll(".copyToClipboard").forEach(
    (button) => button.addEventListener("click", (event) =>
  {
    const element = button.parentNode.querySelector(":scope .authKey");
    element.select();
    element.setSelectionRange(0, 99999);
    document.execCommand("copy");
  }))

  // Formats a future date to a human readable duration
  const toHumanDuration = (date) => {
    const units = ["seconds", "minutes", "hours", "days", "weeks"];
    const steps = [60, 60, 24, 7];
    let index = 0;
    let val = date - Date.now()/1000;

    if (val <= 0)
      return "now";

    while(val > 2*steps[index]) {
      val /= steps[index++];
    }

    if (val < 5)
      val = Math.round(val*5)/5
    else
      val = Math.round(val)

    return `in ${val} ${units[index]}`
  }

  // Augment expire timestamps
  const updateTimestamps = () => {
    document.querySelectorAll("td[data-expire]").forEach((field) => {
      const expiry = field.getAttribute("data-expire");
      if (expiry == "-1")
        return;

      const expires = parseInt(expiry);
      if (!isNaN(expires))
        field.textContent = toHumanDuration(expires);
    });
  }
  setInterval(updateTimestamps, 5000)
  updateTimestamps();
}());