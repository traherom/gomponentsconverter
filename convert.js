document.addEventListener('alpine:init', () => {
  Alpine.data('converter', function() {
    return {
      input: this.$persist(""),
      output: this.$persist(""),

      settings: this.$persist({
        prefix_html: "",
        prefix_core: "",
      }),

      conversion_error: "",
      show_copied: false,
      copied_text: "loading",

      init() {
        if (this.input === "") {
          this.input = initialInput;
        }
        if (this.output === "") {
          this.output = initialOutput;
        }
      },

      imports() {
        const prefixHelper = this.settings.prefix_html === "" ? "." : this.settings.prefix_html;
        const prefixCore = this.settings.prefix_core === "" ? "." : this.settings.prefix_core;
        return `import (\n\t${prefixCore} "maragu.dev/gomponents"\n\t${prefixHelper} "maragu.dev/gomponents/html"\n)`
      },

      async convert() {
        try {
          if (this.input === "") {
            this.output = ""
            return
          }

          const response = await fetch("/convert", {
            method: "POST",
            body: JSON.stringify({
              data: this.input,
              prefix_html: this.settings.prefix_html,
              prefix_core: this.settings.prefix_core,
            }),
          });
          if (!response.ok) {
            throw new Error('Response status: ' + response.status);
          }

          const json = await response.json();
          this.output = json.converted;
          this.conversion_error = json.error;
        } catch (error) {
          console.error(error.message);
          this.conversion_error = error.message;
        }
      },

      async copyImports() {
        this.copy(this.imports());
      },
      async copyOutput() {
        this.copy(this.output);
      },
      async copy(data) {
        this.copied_text = "Copied to clipboard!";
        this.show_copied = true;
        setTimeout(() => {
          this.show_copied = false;
        }, 1000);

        try {
          await navigator.clipboard.writeText(data);
        } catch (err) {
          this.copied_text = `Copy failed: ${err}`;
        }
      },
    }
  })
})

