var App = {
  init: function() {
    App.bindEvents();
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
      .done(App.handleSuccess.bind($this))
      .fail(App.handleFailure);
  },
  handleSuccess: function() {
    this.closest('tr').remove();
  },
  handleFailure(_jqXHR, _x, errorMsg) {
    alert('Oops... Something wrong is not right. ' + errorMsg);
  }
};

$(document).ready(App.init);
