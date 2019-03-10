var DeleteEmail = {
  init: function() {
    DeleteEmail.bindEvents();
  },
  bindEvents: function() {
    $('.delete').click(this.handleDelete);
  },
  handleDelete: function(event) {
    event.preventDefault();

    if (!confirm('Are you sure?')) {
      return;
    }

    $this = $(this);
    var url = $this.attr('href');

    $.ajax({ url: url, type: 'DELETE' })
      .done(DeleteEmail.handleSuccess.bind($this))
      .fail(DeleteEmail.handleFailure);
  },
  handleSuccess: function() {
    this.closest('tr').remove();
  },
  handleFailure(_jqXHR, _, errorMsg) {
    alert('Oops... Something wrong is not right. ' + errorMsg);
  }
};

var ValidateEmail = {
  init: function() {
    this.bindEvents();
  },
  bindEvents: function() {
    $('.validate').click(this.handleValidation);
  },
  handleValidation: function(event) {
    event.preventDefault();

    $this = $(this);
    var url = $this.attr('href');

    $.ajax({
      url: url,
      type: 'GET',
      beforeSend: function() {
        $this.prop('disabled', true).text('Validating...');
      }
    })
      .done(ValidateEmail.handleSuccess.bind($this))
      .fail(ValidateEmail.handleFailure)
      .always(function() {
        $this.prop('enabled', true).text('Validate');
      });
  },
  handleSuccess: function(data) {
    if ('format_valid' in data) {
      this.closest('tr').find('td:eq(2)').text(data.format_valid)
    }
    if ('score' in data) {
      this.closest('tr').find('td:eq(3)').text(data.score)
    }
    if (data.did_you_mean) {
      this.closest('tr').find('td:eq(4)').text(data.did_you_mean)
    }
  },
  handleFailure(_jqXHR, _, errorMsg) {
    alert('Oops... Something wrong is not right. ' + errorMsg);
  }
};

$(document).ready(function() {
  DeleteEmail.init();
  ValidateEmail.init();
});
